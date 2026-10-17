#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Start services in separate terminals:"
echo "  go run ./services/gateway/cmd/gateway"
echo "  go run ./services/moment-service/cmd/moment-service"
echo "  go run ./services/media-service/cmd/media-service"
echo "  go run ./services/render-service/cmd/render-service"
echo "  (cd ml && uv run python -m momentos_ml.worker)"
echo "  pnpm mobile"
