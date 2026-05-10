#!/bin/bash
set -eux

cd $(dirname $0)
cd ../..

docker run --rm \
       -v $PWD:/workdir \
       -w /workdir \
       -u "$(id -u):$(id -g)" \
       openapitools/openapi-generator-cli:v7.11.0 generate \
         -i oas/openapi.yml \
         -g typescript-axios \
         -o oas/ts-axios

mkdir -p editor-ui/src/oapi
mv oas/ts-axios/*.ts editor-ui/src/oapi/
rm -rf oas/ts-axios
