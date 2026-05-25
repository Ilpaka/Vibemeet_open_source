# Architecture

This document covers how Vibemeet is structured: how a request flows from
the browser to LiveKit and back, how the anonymous and authenticated paths
differ, what's stored where, and the rationale behind a few non-obvious
design choices.

## High-level topology

```
       Browser
          │
          │  HTTPS (UI, REST)              WebRTC (UDP, signaling over WS)
          │                                       │
          ▼                                       ▼
       ┌──────┐   reverse proxy           ┌───────────────┐
       │Nginx │ ──────────────────────────│ LiveKit (SFU) │
       └──┬───┘                           └───────────────┘
          │
          │  internal HTTP
          ▼
       ┌────────────┐
       │ Go API     │
       │ (Gin)      │
       └─┬─────┬────┘
         │     │
         │     └────────────────┐
         ▼                      ▼
    ┌──────────┐           ┌────────┐
    │ Postgres │           │ Redis  │
    └──────────┘           └────────┘
```

Nginx terminates HTTP and TLS, serves static assets, and proxies `/api/`,
`/screen-share/`, `/server-info`, and `/health` to the Go backend. LiveKit
runs as a separate service; the browser talks to it directly over
WebSockets for signaling and over UDP for media.

The Go backend is the only component that touches the database. LiveKit
holds no persistent state — it's a media router (SFU) only.

## Two parallel flows: anonymous vs authenticated

Vibemeet was built anonymous-first because the friction of "create an
account to join a meeting" is, in practice, the reason people end up on
proprietary services. Every endpoint required to host or join a room works
without an account.

The price: anonymous rooms are tied to a `participant_id` cookie issued by
the backend on first request (`middleware/participant.go`). Lose the
cookie, lose the identity.

Authentication is layered on top for the things that actually need it:

| Capability                  | Anonymous | Authenticated |
| --------------------------- | --------- | ------------- |
| Create room                 | ✓         | ✓             |
| Join room                   | ✓         | ✓             |
| Send/receive chat in a room | ✓         | ✓             |
| Get a LiveKit token         | ✓         | ✓             |
| List my rooms               | —         | ✓             |
| Generate invite links       | —         | ✓             |
| Persist per-user settings   | —         | ✓             |
| Edit/delete a persisted room| —         | ✓             |

Anonymous rooms live in `anonymous_rooms` / `anonymous_participants` and
expire automatically. Persistent rooms live in `rooms` /
`room_participants`. The schema is intentionally parallel so the same
LiveKit room name space serves both.

## Request flow: anonymous join

1. Browser issues `POST /api/v1/rooms/:id/join` (no `Authorization`
   header). `ParticipantMiddleware` reads or mints the `X-Participant-ID`
   cookie.
2. `AnonymousRoomHandler.Join` validates the room exists in
   `anonymous_rooms`, inserts a row in `anonymous_participants`, and
   returns the room metadata.
3. Browser calls `POST /api/v1/rooms/:id/media/token`. The handler asks
   `AnonymousMediaService` to mint a LiveKit access token signed with
   `LIVEKIT_API_SECRET`, scoped to the room name and the participant ID.
4. Browser opens a WebSocket to `LIVEKIT_FRONTEND_URL`, presents the
   token, and negotiates WebRTC. From this point, media doesn't go through
   the Go backend — it's browser ↔ LiveKit directly.
5. For chat, the browser polls `GET /api/v1/rooms/:id/chat/messages`.
   Messages are stored in a Redis sorted set with a 6-hour TTL — fast to
   read, no schema migration when fields change, and the data evaporates
   when the room is over.

## Request flow: authenticated user

1. `POST /api/v1/auth/login` hits `AuthService.Login` → bcrypt password
   verification → issues an access JWT (15 min default) and a refresh JWT
   (7 days), persisting the SHA256 hash of the refresh token in
   `user_sessions`.
2. Subsequent requests carry `Authorization: Bearer <access>`. The
   `AuthMiddleware` decodes the JWT, verifies the signature with
   `JWT_ACCESS_SECRET`, and injects the user ID into the Gin context.
3. `POST /api/v1/auth/refresh` rotates the refresh token: the old hash is
   marked revoked, a new pair is issued and persisted.

Logout-everywhere is just a `DELETE` over `user_sessions WHERE user_id = ?`
— stateless access tokens become unusable once their 15-minute window
expires.

## Why LiveKit?

Building an SFU is a multi-year project. LiveKit is open source, runs as
a single container, has a stable Go server-side API for token minting, and
its client SDK handles the painful parts of WebRTC (codec negotiation,
adaptive bitrate, simulcast, reconnects).

The backend never sees media traffic. It mints scoped JWTs and trusts
LiveKit to enforce them. This means horizontal scaling of the API is
trivial — every API node is stateless except for the shared Postgres /
Redis backplane.

For server-side screen sharing (the `/screen-share/` endpoint group), the
backend uses Pion directly. This path is opt-in and exists for the case
where a participant wants to broadcast the server's screen rather than
their own (kiosk and demo scenarios).

## Data model

Important tables (full DDL in `init_db.sql`):

- `users`, `user_sessions`, `user_settings` — accounts, refresh tokens,
  per-user device defaults.
- `rooms`, `room_participants`, `room_invites` — persistent rooms.
- `chat_messages` — chat for persistent rooms.
- `participant_stats` — connection quality snapshots (filled from LiveKit
  telemetry; consumed by the `/stats` endpoints).
- `audit_log` — append-only event log.
- `anonymous_rooms`, `anonymous_participants` — ephemeral room state.

Anonymous chat lives entirely in Redis sorted sets keyed by
`chat:room:<room-id>:messages`, scored by `created_at` in milliseconds.

## Code organization

The backend follows a standard handler → service → repository split:

- **Handlers** (`internal/handler`) deal with HTTP. They parse, validate,
  delegate, and serialize. They do not query the database.
- **Services** (`internal/service`) hold business logic. They depend on
  repository interfaces — not on `*pgxpool.Pool` directly — so they're
  easy to test.
- **Repositories** (`internal/repository`) are the only layer that knows
  about Postgres and Redis.
- **Domain** (`internal/domain`) defines the shared types and interface
  contracts.

Cross-cutting concerns (auth, rate limiting, participant identity, CORS,
request logging, error mapping) live in `internal/middleware` as Gin
handlers.

## Configuration

Everything is environment-driven (`internal/config/config.go`). The
runtime reads `.env` if present, then overlays real environment
variables, then validates that critical secrets are non-empty. There is no
fallback to insecure defaults in `validate()` for JWT secrets or the
database DSN — the process will refuse to start without them.

## Graceful shutdown

`cmd/server/main.go` listens for `SIGINT` / `SIGTERM` and drains the HTTP
server with a 10-second timeout. In-flight requests get to finish; pgx
and Redis pools are closed via `defer`. Containers receive SIGTERM on
`docker compose down`, so no special wiring is needed.

## Trade-offs you'll notice in the code

- **Chat as polling, not WebSockets.** A WebSocket chat handler existed
  in an earlier revision; it was removed because polling against Redis is
  cheap, the room sizes targeted (≤ 50 participants) don't justify a
  per-room socket fan-out, and removing a moving part is its own reward.
- **Anonymous chat in Redis instead of Postgres.** Anonymous rooms are
  transient. Keeping their chat in a TTL'd Redis structure means no
  cleanup job, no schema growth, and message reads cost one ZRANGE.
- **Two parallel handler trees** (`Room` + `AnonymousRoom`, `Chat` +
  `AnonymousChat`) instead of an `if userID == nil` branch in one handler.
  The duplication is small and the two flows have different auth
  semantics; keeping them apart makes both easier to reason about.
