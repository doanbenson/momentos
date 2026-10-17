#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPEC="$ROOT/packages/contracts/openapi.yaml"

echo "OpenAPI source: $SPEC"
echo "TODO: wire openapi-generator / oapi-codegen / openapi-python-client"
echo "  → packages/sdk-ts"
echo "  → services/*/internal/api (optional)"
echo "  → ml/momentos_ml/contracts"
