#!/bin/bash
set -eux

cd $(dirname $0)
cd ../..

OUT=${1:-oas/redoc/index.html}
npx @redocly/cli build-docs oas/openapi.yml -o "$OUT" \
  --theme.openapi.schemaDefinitionsTagName=Models
