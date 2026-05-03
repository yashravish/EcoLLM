#!/usr/bin/env bash
# Seed the development database with model registry entries and a demo org.
# Requires: psql in PATH or a running postgres Docker container.
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://ecollm:ecollm_dev@localhost:5432/ecollm?sslmode=disable}"

echo "Seeding database..."
psql "$DATABASE_URL" -f "$(dirname "$0")/../db/seeds/models.sql"
echo "Seed complete."
