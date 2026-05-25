# Vibemeet

[![CI](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/ci.yml)
[![lint](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/lint.yml)
[![compose](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/compose.yml/badge.svg?branch=main)](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/compose.yml)
[![migrate](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/migrate.yml/badge.svg?branch=main)](https://github.com/Ilpaka/Vibemeet_open_source/actions/workflows/migrate.yml)
[![Go](https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white)](go.mod)
[![LiveKit](https://img.shields.io/badge/livekit-SFU-FF4D4D)](https://livekit.io)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> **Техническая спецификация на русском →** [`docs/SPEC.ru.md`](docs/SPEC.ru.md)
> &nbsp;·&nbsp; English docs: [Architecture](docs/ARCHITECTURE.md) · [API](docs/API.md) · [Deployment](docs/DEPLOYMENT.md)

Self-hostable video conferencing built on Go and LiveKit. Drop in your own
infrastructure, get a Zoom-style web client with rooms, chat, and screen
sharing - no third-party meeting service in the loop.

The default flow is anonymous: anyone with a room link can join from the
browser without an account. Optional authentication unlocks room ownership,
invites, and per-user settings.

```
┌────────┐   HTTPS    ┌────────┐   HTTP    ┌────────┐   SQL    ┌──────────┐
│ Browser│ ────────▶  │ Nginx  │ ────────▶ │ Go API │ ───────▶ │ Postgres │
└────┬───┘            └────────┘           └───┬────┘          └──────────┘
     │                                         │  ┌───────┐
     │                                         └─▶│ Redis │  ← chat, rate limits
     │   WebRTC (UDP)                             └───────┘
     └──────────────────────────▶  ┌──────────┐
                                   │ LiveKit  │  ← SFU: media routing
                                   └──────────┘
```

## Features

- WebRTC video and audio for any number of participants per room.
- Screen sharing - both browser-native (LiveKit track) and server-side
  capture via Pion WebRTC.
- Persistent text chat for authenticated rooms; ephemeral Redis-backed chat
  (6-hour TTL) for anonymous rooms.
- Anonymous rooms with cookie-based participant identity - no signup
  required to host or join.
- JWT auth (access + refresh, refresh tokens hashed and stored in Postgres
  so they can be revoked).
- Per-IP rate limiting on auth endpoints, backed by Redis.
- Health and server-info endpoints suitable for load balancers and the
  frontend to discover the LiveKit URL.
- Graceful shutdown, structured logging (`log/slog`), env-driven config
  with validation.

## Stack

| Layer        | Technology                                       |
| ------------ | ------------------------------------------------ |
| API server   | Go 1.25, [Gin](https://github.com/gin-gonic/gin) |
| Database     | PostgreSQL 15 (via `pgx/v5`)                     |
| Cache / chat | Redis 7                                          |
| Media server | [LiveKit](https://livekit.io) (self-hosted SFU)  |
| WebRTC       | [Pion](https://github.com/pion/webrtc) (server-side screen capture) |
| Frontend     | Static HTML + vanilla JS + LiveKit client SDK    |
| Reverse proxy| Nginx                                            |
| Packaging    | Docker, Docker Compose                           |

## Quick start

You need Docker and Docker Compose v2.

```bash
git clone <your-fork> vibemeet
cd vibemeet
cp env.sample .env
./start.sh
```

Open <http://localhost> and create a room. To join from another device on
the same LAN, set `HOST_IP` in `.env` to the host machine's LAN address
(for example `192.168.1.42`) before starting - LiveKit needs that IP to
advertise correct ICE candidates.

Stop the stack with `docker compose --env-file .env down`.

## Configuration

All runtime configuration is environment-driven. `env.sample` lists every
variable with development defaults; `env.prod.example` is the production
template.

Critical variables:

| Variable               | Purpose                                              |
| ---------------------- | ---------------------------------------------------- |
| `POSTGRES_PASSWORD`    | Database password (required).                        |
| `JWT_ACCESS_SECRET`    | HMAC secret for access tokens. Use `openssl rand -hex 32`. |
| `JWT_REFRESH_SECRET`   | HMAC secret for refresh tokens. Use `openssl rand -hex 32`. |
| `LIVEKIT_API_KEY`      | LiveKit API key (shared with the LiveKit container). |
| `LIVEKIT_API_SECRET`   | LiveKit API secret.                                  |
| `HOST_IP`              | Host LAN/public IP for LiveKit ICE candidates.       |
| `LIVEKIT_FRONTEND_URL` | Public WebSocket URL clients use to reach LiveKit.   |

Sensible defaults are baked in for `POSTGRES_DB`, `POSTGRES_USER`, ports,
and timeouts. See `internal/config/config.go`.

## Project layout

```
cmd/server/             Process entrypoint, subcommands, router wiring
internal/
  config/               Env loading and validation
  domain/               Core types (users, rooms, chat, stats)
  handler/              HTTP handlers (Gin)
  middleware/           Auth, CORS, rate limiting, request logging
  migration/            Goose migrations (embedded into the binary)
  repository/           Postgres + Redis persistence
  service/              Business logic (auth, rooms, LiveKit, screen share)
pkg/
  jwt/                  Token signing and validation
  logger/               slog wrapper
web/                    Static frontend (HTML/CSS/JS)
docs/                   Architecture, API reference, deployment guide, spec
docker-compose.yml      Local development stack
docker-compose.prod.yml Production stack
livekit{.prod}.yaml     LiveKit server config
nginx{.prod}.conf       Reverse proxy config
```

## Database migrations

Schema is managed with [goose](https://github.com/pressly/goose) and embedded
into the server binary. On startup the server applies every pending
migration before opening the HTTP listener (set `MIGRATE_ON_BOOT=false` to
opt out and run them as a separate CI/CD step).

```bash
make migrate-up        # apply all pending migrations
make migrate-status    # show applied/pending state
make migrate-version   # print the current schema version
make migrate-down      # roll back the most recent migration
make migrate-redo      # roll back and re-apply the latest migration
```

These targets delegate to the same `vibemeet migrate <cmd>` subcommand the
binary exposes, so the production image can run them without a separate
goose CLI.

Migration files live in [internal/migration/](internal/migration/). New
migrations are sequential SQL files (`00002_*.sql`, `00003_*.sql`, ...) with
goose `-- +goose Up` / `-- +goose Down` markers.

## Local development without Docker

You can run the API against your host's Postgres, Redis, and LiveKit
instances if you'd rather skip the full Compose stack.

```bash
# 1. Have Postgres, Redis, and LiveKit running locally.
# 2. Export config (DATABASE_DSN, REDIS_ADDR, JWT secrets, ...).
export $(grep -v '^#' .env | xargs)
# 3. Run. Migrations are applied automatically on first start.
make run
# or:  go run ./cmd/server
```

The server listens on `SERVER_PORT` (8080 by default). The static frontend
in `web/` expects to be served by Nginx in front of the API, but you can
also open the HTML files directly during development - they'll talk to the
API via CORS.

Useful Make targets:

```
make help     # list everything
make build    # build the server binary into bin/server
make run      # go run ./cmd/server
make test     # go test ./...
make up       # docker compose up (with .env)
make down     # docker compose down
```

## Production deployment

See `docs/DEPLOYMENT.md` for the full procedure. In short:

```bash
cp env.prod.example .env.prod
# Fill in HOST_IP, LIVEKIT_FRONTEND_URL, every secret, etc.
./deploy.sh
```

Production uses non-standard ports (HTTP 18080, LiveKit 17880/17881/17882)
so it can coexist with anything already bound to 80/443/7880 on the host.
Terminate TLS at Nginx (mount certificates into `./ssl/`) or behind your
own reverse proxy.

## API

A complete endpoint reference lives in `docs/API.md`. Highlights:

- `POST /api/v1/auth/{register,login,refresh}` - user accounts.
- `POST /api/v1/rooms` - create an anonymous room (no auth).
- `POST /api/v1/rooms/:id/join`, `/leave`, `/media/token` - anonymous flow.
- `GET/POST /api/v1/rooms/:id/chat/messages` - anonymous chat.
- `GET /api/v1/users/me`, `/settings` - authenticated user features.
- `GET /server-info` - frontend discovery for `HOST_IP` and LiveKit URL.

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - request flow, anonymous
  vs authenticated paths, data model, LiveKit integration.
- [docs/API.md](docs/API.md) - REST API reference.
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) - production deployment, TLS,
  firewall rules.
- [docs/SPEC.ru.md](docs/SPEC.ru.md) - техническая спецификация (RU):
  цели и нецели, NFR, модель угроз, компромиссы, дорожная карта.

## License

MIT.
