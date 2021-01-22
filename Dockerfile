FROM golang:1.10-alpine

ARG git_tag
ARG git_commit

RUN apk add --no-cache git build-base curl
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/github.com/ory/keto

ADD ./Gopkg.lock ./Gopkg.lock
ADD ./Gopkg.toml ./Gopkg.toml
RUN dep ensure -vendor-only

ADD . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -X gitlab.host1plus.com/linas/keto/cmd.Version=$git_tag -X gitlab.host1plus.com/linas/keto/cmd.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` -X gitlab.host1plus.com/linas/keto/cmd.GitHash=$git_commit" -a -installsuffix cgo -o keto

FROM scratch

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /go/src/gitlab.host1plus.com/linas/keto/keto /usr/bin/keto

ENTRYPOINT ["keto"]

CMD ["serve"]
