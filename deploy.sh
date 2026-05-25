#!/usr/bin/env bash
# Deploy the Vibemeet production stack.
# Requires .env.prod (copy from env.prod.example and fill in values).

set -euo pipefail

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  echo "error: docker and docker compose plugin are required"
  exit 1
fi

if [[ ! -f .env.prod ]]; then
  echo "error: .env.prod not found. Create one with: cp env.prod.example .env.prod"
  exit 1
fi

echo "Bringing up production stack..."
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build

sleep 10
docker compose -f docker-compose.prod.yml --env-file .env.prod ps

cat <<EOF

Vibemeet (prod) is running.
  Web UI:   http://<HOST_IP>:\${HTTP_PORT:-18080}
  Logs:     docker compose -f docker-compose.prod.yml --env-file .env.prod logs -f
  Stop:     docker compose -f docker-compose.prod.yml --env-file .env.prod down
EOF
