#!/bin/bash

set -eu

cd $(dirname $0);
cd ../..

mkdir -p ./internal/oapi

go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oas/config/models.yml oas/openapi.yml
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oas/config/server.yml oas/openapi.yml
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oas/config/spec.yml oas/openapi.yml
