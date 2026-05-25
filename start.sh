#!/usr/bin/env bash
# Launch the Vibemeet stack for local development.
# Requires Docker and a populated .env file (copy from env.sample).

set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is not installed"
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "error: docker compose plugin is not available"
  exit 1
fi

if [[ ! -f .env ]]; then
  echo "error: .env not found. Create one with: cp env.sample .env"
  exit 1
fi

echo "Starting Vibemeet (dev stack)..."
docker compose --env-file .env up -d --build

echo
docker compose --env-file .env ps

cat <<EOF

Vibemeet is starting up.
  Web UI:   http://localhost
  Health:   http://localhost/health

Logs:       docker compose --env-file .env logs -f
Stop:       docker compose --env-file .env down
EOF
