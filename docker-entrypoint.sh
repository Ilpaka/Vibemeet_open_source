#!/bin/sh
set -e

echo "[entrypoint] waiting for postgres..."
until nc -z postgres 5432; do
  sleep 0.2
done
echo "[entrypoint] postgres is up"

echo "[entrypoint] waiting for redis..."
until nc -z redis 6379; do
  sleep 0.2
done
echo "[entrypoint] redis is up"

echo "[entrypoint] starting vibemeet server"
exec /app/server
