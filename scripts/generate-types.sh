#!/usr/bin/env bash
# Generate TypeScript types from the OpenAPI spec.
# Requires: npx (Node 18+)
set -euo pipefail

SPEC="packages/contracts/openapi.yaml"
OUT="apps/web/src/types/api.ts"

if [ ! -f "$SPEC" ]; then
  echo "ERROR: OpenAPI spec not found at $SPEC" >&2
  exit 1
fi

echo "Generating TypeScript types from $SPEC..."
mkdir -p "$(dirname "$OUT")"
npx --yes openapi-typescript "$SPEC" -o "$OUT"
echo "Generated: $OUT"
