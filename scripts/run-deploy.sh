#!/bin/bash

set -euo pipefail

gox -ldflags "-X gitlab.host1plus.com/linas/keto/cmd.Version=`git describe --tags` -X gitlab.host1plus.com/linas/keto/cmd.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` -X gitlab.host1plus.com/linas/keto/cmd.GitHash=`git rev-parse HEAD`" -output "dist/{{.Dir}}-{{.OS}}-{{.Arch}}";
npm version -f --no-git-tag-version $(git describe --tag);
