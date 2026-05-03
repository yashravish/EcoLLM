#!/usr/bin/env bash
# Run database migrations using golang-migrate via Docker.
# Usage: bash scripts/migrate.sh [up|down|version] [N]
set -euo pipefail

DIRECTION="${1:-up}"
STEPS="${2:-}"
DATABASE_URL="${DATABASE_URL:-postgres://ecollm:ecollm_dev@localhost:5432/ecollm?sslmode=disable}"
MIGRATIONS_DIR="$(cd "$(dirname "$0")/.." && pwd)/db/migrations"

CMD_ARGS=("-path" "/migrations" "-database" "$DATABASE_URL" "$DIRECTION")
if [ -n "$STEPS" ]; then
  CMD_ARGS+=("$STEPS")
fi

echo "Running migration: $DIRECTION ${STEPS:-}"
docker run --rm --network host \
  -v "$MIGRATIONS_DIR:/migrations:ro" \
  migrate/migrate:v4.17.0 \
  "${CMD_ARGS[@]}"
echo "Done."
