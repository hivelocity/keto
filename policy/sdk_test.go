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
 * @copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */

package policy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/ory/herodot"
	. "github.com/hivelocity/keto/policy"
	keto "github.com/hivelocity/keto/sdk/go/keto/swagger"
	"github.com/hivelocity/ladon"
	"github.com/hivelocity/ladon/manager/memory"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockPolicy(t *testing.T) keto.Policy {
	originalPolicy := &ladon.DefaultPolicy{
		ID:          uuid.New(),
		Description: "description",
		Subjects:    []string{"<peter>"},
		Effect:      ladon.AllowAccess,
		Resources:   []string{"<article|user>"},
		Actions:     []string{"view"},
		Conditions: ladon.Conditions{
			"ip": &ladon.CIDRCondition{
				CIDR: "1234",
			},
			"owner": &ladon.EqualsSubjectCondition{},
		},
	}
	out, err := json.Marshal(originalPolicy)
	require.NoError(t, err)

	var apiPolicy keto.Policy
	require.NoError(t, json.Unmarshal(out, &apiPolicy))
	out, err = json.Marshal(&apiPolicy)
	require.NoError(t, err)

	var checkPolicy ladon.DefaultPolicy
	require.NoError(t, json.Unmarshal(out, &checkPolicy))
	require.EqualValues(t, checkPolicy.Conditions["ip"], originalPolicy.Conditions["ip"])
	require.EqualValues(t, checkPolicy.Conditions["owner"], originalPolicy.Conditions["owner"])

	return apiPolicy
}

func TestPolicySDK(t *testing.T) {
	handler := &Handler{
		Manager: &memory.MemoryManager{Policies: map[string]ladon.Policy{}},
		H:       herodot.NewJSONWriter(nil),
	}

	router := httprouter.New()
	handler.SetRoutes(router)
	server := httptest.NewServer(router)

	client := keto.NewPolicyApiWithBasePath(server.URL)

	p := mockPolicy(t)

	t.Run("TestPolicyManagement", func(t *testing.T) {
		_, response, err := client.GetPolicy(p.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, response.StatusCode)

		result, response, err := client.CreatePolicy(p)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, response.StatusCode)
		assert.EqualValues(t, p, *result)

		result, response, err = client.GetPolicy(p.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.EqualValues(t, p, *result)

		p.Subjects = []string{"stan"}
		result, response, err = client.UpdatePolicy(p.Id, p)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.EqualValues(t, p, *result)

		results, response, err := client.ListPolicies(0, 10)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Len(t, results, 1)

		results, response, err = client.ListPolicies(10, 1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Len(t, results, 0)

		result, response, err = client.GetPolicy(p.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.EqualValues(t, p, *result)

		response, err = client.DeletePolicy(p.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, response.StatusCode)

		_, response, err = client.GetPolicy(p.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
	})
}
