#!/usr/bin/env sh
set -e

# If you want to run this before pushing
# mkdir -p /opt/resource
# chown -R yourUser:yourUser /opt/resource
# go build -o /opt/resource/out cmd/out/out.go
# export CF_API=google-api
# export CF_USERNAME=engineering-enablement user
# export CF_PASSWORD=asdasd
# export CF_ORG=engineering-enablement
# export CF_SPACE=integration_test

go run .integration_test/integration.go `pwd`
