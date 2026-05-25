# Deployment

This guide covers a single-host production deployment using
`docker-compose.prod.yml`. The same stack runs comfortably on a 2 vCPU /
4 GB VM for tens of concurrent participants; scale LiveKit horizontally if
you need more.

## Prerequisites

- A Linux host with a public IP (or a private IP if you'll be behind
  another reverse proxy).
- Docker 24+ and the Docker Compose plugin (`docker compose version`).
- Open inbound ports - see the firewall section below.
- TLS certificates if you intend to serve over HTTPS (you should).

## 1. Configure

```bash
git clone <your-fork> /opt/vibemeet
cd /opt/vibemeet
cp env.prod.example .env.prod
```

Edit `.env.prod` and set every value:

```bash
HOST_IP=203.0.113.10
LIVEKIT_FRONTEND_URL=wss://meet.example.com:17880

POSTGRES_DB=vibemeet
POSTGRES_USER=vibemeet
POSTGRES_PASSWORD=$(openssl rand -hex 24)

JWT_ACCESS_SECRET=$(openssl rand -hex 32)
JWT_REFRESH_SECRET=$(openssl rand -hex 32)

LIVEKIT_API_KEY=$(openssl rand -hex 8)
LIVEKIT_API_SECRET=$(openssl rand -hex 24)
```

`HOST_IP` is what LiveKit advertises in ICE candidates. It must be
reachable from every client that connects.

`LIVEKIT_FRONTEND_URL` is what the browser uses. Use `wss://` if you
terminate TLS in front of LiveKit, `ws://` otherwise.

## 2. (Recommended) Terminate TLS

Place certificates under `./ssl/`:

```
ssl/
├── server.crt
└── server.key
```

`docker-compose.prod.yml` mounts that directory into the Nginx container.
Update `nginx.prod.conf` to add an HTTPS server block listening on 443
and proxy to `backend:8080` exactly like the existing HTTP block.

If you're behind another reverse proxy (Caddy, Cloudflare, Traefik),
leave Nginx on plain HTTP and terminate TLS upstream.

## 3. Launch

```bash
./deploy.sh
```

Equivalent manual invocation:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

Verify the stack is healthy:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
curl http://localhost:18080/health
```

## Port reference

| Port          | Protocol | Purpose                          |
| ------------- | -------- | -------------------------------- |
| 18080         | TCP      | Nginx HTTP                       |
| 18443         | TCP      | Nginx HTTPS                      |
| 17880         | TCP      | LiveKit WebSocket signaling      |
| 17881         | TCP      | LiveKit TCP media fallback       |
| 17882         | UDP      | LiveKit WebRTC                   |
| 50000–50050   | UDP      | LiveKit RTP media range          |

Ports are chosen to coexist with anything already bound to the standard
80/443/7880 on the host.

## Firewall

UFW:

```bash
ufw allow 18080/tcp
ufw allow 18443/tcp
ufw allow 17880/tcp
ufw allow 17881/tcp
ufw allow 17882/udp
ufw allow 50000:50050/udp
```

firewalld:

```bash
firewall-cmd --permanent --add-port=18080/tcp
firewall-cmd --permanent --add-port=18443/tcp
firewall-cmd --permanent --add-port=17880/tcp
firewall-cmd --permanent --add-port=17881/tcp
firewall-cmd --permanent --add-port=17882/udp
firewall-cmd --permanent --add-port=50000-50050/udp
firewall-cmd --reload
```

If you run a cloud VM, mirror the same rules in the provider's security
groups. UDP for media is non-negotiable - TCP fallback exists but is
substantially worse for real-time video.

## Operations

```bash
COMPOSE="docker compose -f docker-compose.prod.yml --env-file .env.prod"

$COMPOSE logs -f                # tail all logs
$COMPOSE logs -f backend        # only the API
$COMPOSE restart backend        # rolling restart
$COMPOSE up -d --build backend  # rebuild + redeploy the API
$COMPOSE down                   # stop everything (volumes persist)
$COMPOSE down -v                # nuke volumes too (data loss!)
```

## Updates

```bash
cd /opt/vibemeet
git pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

### Database migrations

Schema is managed with [goose](https://github.com/pressly/goose) and embedded
into the server binary. By default migrations are applied automatically on
startup; new pending migrations apply before the HTTP listener starts.

To manage migrations out-of-band (recommended for CI/CD pipelines), set
`MIGRATE_ON_BOOT=false` in `.env.prod` and run them explicitly:

```bash
# Inside the running container or with DATABASE_DSN exported:
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec vibemeet ./server migrate status
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec vibemeet ./server migrate up
```

Available subcommands: `up`, `up-by-one`, `down`, `redo`, `status`, `version`.

To roll a release back, deploy the previous image and run
`./server migrate down` to revert the most recent migration.

## Backups

The Postgres data volume is `postgres_data` (Docker named volume). Snapshot
it the same way you snapshot any other Postgres deployment:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" \
  | gzip > "vibemeet-$(date +%F).sql.gz"
```

Redis state is ephemeral (anonymous chat messages with a 6-hour TTL); no
backup needed.

## Troubleshooting

**Browser shows "connecting to room" forever.** Almost always a LiveKit
reachability issue. Confirm:

- The browser can reach `LIVEKIT_FRONTEND_URL` (try opening it directly -
  you should see "OK").
- UDP ports 17882 and 50000–50050 are open.
- `HOST_IP` matches the address the browser uses to reach the server.

**`vibemeet` container exits with "Database migrations failed".** The
volume has data that conflicts with a pending migration. Use
`./server migrate status` to inspect which migration is stuck, fix the
underlying conflict (often a manual schema change), or run
`docker compose down -v` (destructive) and start fresh.

**Backend logs "JWT secrets must be set".** Your `.env.prod` is empty or
not being loaded. Confirm `--env-file .env.prod` is on every compose
invocation.

**LiveKit logs "could not establish ICE".** `HOST_IP` is wrong, a NAT/UFW
rule is blocking UDP, or the client is behind symmetric NAT. Add a TURN
server for the last case.
