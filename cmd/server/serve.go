/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @Copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 *
 */

package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/ory/fosite"
	"github.com/ory/go-convenience/corsx"
	"github.com/ory/go-convenience/stringsx"
	"github.com/ory/graceful"
	"github.com/ory/herodot"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"github.com/hivelocity/keto/authentication"
	"github.com/hivelocity/keto/health"
	"github.com/hivelocity/keto/policy"
	"github.com/hivelocity/keto/role"
	"github.com/hivelocity/keto/warden"
	"github.com/hivelocity/ladon"

	negronilogrus "github.com/meatballhat/negroni-logrus"
)

// RunServe runs the Keto API HTTP server
func RunServe(
	logger *logrus.Logger,
	buildVersion, buildHash string, buildTime string,
) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		router := httprouter.New()

		m, err := newManagers(viper.GetString("DATABASE_URL"), logger)
		if err != nil {
			logger.
				WithError(err).
				Fatal("Unable to initialise backends")
		}

		var strategy fosite.ScopeStrategy
		switch viper.GetString("AUTHENTICATOR_OAUTH2_INTROSPECTION_SCOPE_STRATEGY") {
		case "hierarchic":
			strategy = fosite.HierarchicScopeStrategy
			break
		case "exact":
			strategy = fosite.ExactScopeStrategy
			break
		case "wildcard":
			fallthrough
		default:
			strategy = fosite.WildcardScopeStrategy
		}

		authenticators := map[string]authentication.Authenticator{
			"subjects": authentication.NewPlaintextAuthentication(),
			"oauth2/access-tokens": authentication.NewOAuth2IntrospectionAuthentication(
				viper.GetString("AUTHENTICATOR_OAUTH2_INTROSPECTION_CLIENT_ID"),
				viper.GetString("AUTHENTICATOR_OAUTH2_INTROSPECTION_CLIENT_SECRET"),
				viper.GetString("AUTHENTICATOR_OAUTH2_INTROSPECTION_TOKEN_URL"),
				viper.GetString("AUTHENTICATOR_OAUTH2_INTROSPECTION_URL"),
				stringsx.Splitx(viper.GetString("AUTHENTICATOR_OAUTH2_INTROSPECTION_SCOPE"), ","),
				strategy,
			),
			"oauth2/clients": authentication.NewOAuth2ClientCredentialsAuthentication(
				viper.GetString("AUTHENTICATOR_OAUTH2_CLIENT_CREDENTIALS_TOKEN_URL"),
			),
		}

		decider := &ladon.Ladon{
			Manager:     m.policyManager,
			AuditLogger: &warden.AuditLoggerLogrus{Logger: logger},
			Matcher:     ladon.DefaultMatcher,
		}
		firewall := warden.NewWarden(decider, m.roleManager, logger)
		writer := herodot.NewJSONWriter(logger)
		roleHandler := role.NewHandler(m.roleManager, writer)
		policyHandler := policy.NewHandler(m.policyManager, writer)
		wardenHandler := warden.NewHandler(writer, firewall, authenticators)
		healthHandler := health.NewHandler(writer, buildVersion, m.readyCheckers)

		roleHandler.SetRoutes(router)
		policyHandler.SetRoutes(router)
		wardenHandler.SetRoutes(router)
		healthHandler.SetRoutes(router)

		n := negroni.New()
		n.Use(negronilogrus.NewMiddlewareFromLogger(logger, "keto"))

		var c http.Handler = n
		if viper.GetString("CORS_ENABLED") == "true" {
			logger.Info("Enabled CORS")
			c = cors.New(corsx.ParseOptions()).Handler(n)
		}

		n.UseHandler(router)

		cert, err := getTLSCertAndKey()
		if err != nil {
			logger.Fatalf("%v", err)
		}

		certs := []tls.Certificate{}
		if cert != nil {
			certs = append(certs, *cert)
		}

		addr := fmt.Sprintf("%s:%s", viper.GetString("HOST"), viper.GetString("PORT"))
		server := graceful.WithDefaults(&http.Server{
			Addr:         addr,
			Handler:      c,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
			TLSConfig: &tls.Config{
				Certificates: certs,
			},
		})

		if err := graceful.Graceful(func() error {
			if cert != nil {
				logger.Printf("Listening on https://%s.\n", addr)
				return server.ListenAndServeTLS("", "")
			}
			logger.Printf("Listening on http://%s.\n", addr)
			return server.ListenAndServe()
		}, server.Shutdown); err != nil {
			logger.Fatalf("Unable to gracefully shutdown HTTP(s) server because %v.\n", err)
			return
		}
	}
}
