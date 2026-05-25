#!/bin/sh
set -e

echo "[nginx] waiting for backend..."
MAX_RETRIES=60
RETRY=0

while [ "$RETRY" -lt "$MAX_RETRIES" ]; do
  if nc -z backend 8080 2>/dev/null; then
    echo "[nginx] backend is up"
    break
  fi
  RETRY=$((RETRY + 1))
  if [ $((RETRY % 5)) -eq 0 ]; then
    echo "[nginx] still waiting... ($RETRY/$MAX_RETRIES)"
  fi
  sleep 1
done

if [ "$RETRY" -eq "$MAX_RETRIES" ]; then
  echo "[nginx] backend port check failed after $MAX_RETRIES attempts - starting anyway"
fi

exec nginx -g "daemon off;"
