#!/usr/bin/env bash
# Initial developer environment setup.
# Run once after cloning: bash scripts/setup.sh
set -euo pipefail

CYAN='\033[0;36m'; RESET='\033[0m'
info() { echo -e "${CYAN}→ $*${RESET}"; }

# 1. Copy .env.example if .env doesn't exist
if [ ! -f .env ]; then
  info "Creating .env from .env.example"
  cp .env.example .env
  echo "  Edit .env with your secrets before running 'make dev'"
else
  info ".env already exists, skipping"
fi

# 2. Check required tools
REQUIRED=(docker go node npm)
for tool in "${REQUIRED[@]}"; do
  if ! command -v "$tool" &>/dev/null; then
    echo "ERROR: '$tool' is required but not found in PATH" >&2
    exit 1
  fi
done
info "All required tools present"

# 3. Install golangci-lint if absent
if ! command -v golangci-lint &>/dev/null; then
  info "Installing golangci-lint..."
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
    sh -s -- -b "$(go env GOPATH)/bin" v1.57.2
fi

# 4. Install Go module dependencies
info "Downloading Go modules..."
(cd apps/api && go mod download)
(cd apps/inference-gateway && go mod download)
(cd apps/carbon-service && go mod download)

# 5. Install Node dependencies
if [ -d apps/web ]; then
  info "Installing Node dependencies..."
  (cd apps/web && npm ci --prefer-offline 2>/dev/null || npm install)
fi

# 6. Pull Docker images in background
info "Pulling Docker images (background)..."
docker compose -f infra/docker/docker-compose.yml pull --quiet &

echo ""
info "Setup complete. Next steps:"
echo "  1. Edit .env with your configuration"
echo "  2. Run 'make dev' to start the stack"
echo "  3. Run 'make migrate' to apply database migrations"
echo "  4. Run 'make seed' to load development data"
