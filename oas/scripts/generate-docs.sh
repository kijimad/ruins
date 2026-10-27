#!/bin/bash
set -eux

cd "$(dirname "$0")"
cd ../..

OUT=${1:-oas/redoc/index.html}
npx @redocly/cli@2.71.2 build-docs oas/openapi.yml -o "$OUT" \
	--openapi.schemaDefinitionsTagName=Models
