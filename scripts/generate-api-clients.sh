#!/bin/sh
set -eu
npx --yes openapi-typescript@7.10.1 openapi.yaml -o src/api.generated.ts
npx --yes @openapitools/openapi-generator-cli@2.18.4 generate -i openapi.yaml -g kotlin -o clients/android-kotlin --additional-properties=library=jvm-retrofit2,serializationLibrary=gson,packageName=app.hwatch.client
