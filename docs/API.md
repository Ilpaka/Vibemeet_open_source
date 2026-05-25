# API reference

All endpoints are served under `/api/v1` unless noted otherwise. JSON in,
JSON out. Errors use the shape `{"error": "human-readable message"}`.

Authenticated endpoints expect `Authorization: Bearer <access_token>`.
Anonymous endpoints expect (and will set) an `X-Participant-ID` cookie.

## Health

### `GET /health`

Liveness probe. Returns `{"status":"ok"}`.

### `GET /server-info`

Used by the frontend to discover the LiveKit URL and host IP:

```json
{
  "host_ip": "192.168.1.42",
  "livekit_port": "7880",
  "livekit_url": "ws://192.168.1.42:7880",
  "environment": "development"
}
```

## Authentication

### `POST /api/v1/auth/register`

```json
{ "email": "user@example.com", "password": "pass", "display_name": "Sam" }
```

Returns `{"user": {...}, "access_token": "...", "refresh_token": "..."}`.
Rate limited per IP.

### `POST /api/v1/auth/login`

```json
{ "email": "user@example.com", "password": "pass" }
```

Same response shape as `/register`. Rate limited.

### `POST /api/v1/auth/refresh`

```json
{ "refresh_token": "..." }
```

Returns a new access/refresh pair. The previous refresh token is revoked.

## Anonymous rooms

These endpoints require no auth but accept (and set) `X-Participant-ID`.

### `POST /api/v1/rooms`

Create an anonymous room.

```json
{ "title": "Standup", "max_participants": 10 }
```

Response:

```json
{
  "id": "uuid",
  "livekit_room_name": "vibemeet-...",
  "title": "Standup",
  "max_participants": 10,
  "status": "active"
}
```

### `GET /api/v1/rooms/:id`

Fetch room metadata by ID.

### `POST /api/v1/rooms/:id/join`

```json
{ "display_name": "Sam" }
```

Records the participant. Returns the updated room and the participant ID.

### `POST /api/v1/rooms/:id/leave`

Marks the current participant as having left.

### `GET /api/v1/rooms/:id/participants`

Returns the list of currently joined participants.

### `POST /api/v1/rooms/:id/media/token`

```json
{ "display_name": "Sam" }
```

Returns a LiveKit access token scoped to the room:

```json
{
  "token": "eyJ...",
  "livekit_url": "ws://host:7880",
  "room_name": "vibemeet-..."
}
```

## Anonymous chat

Backed by Redis with a 6-hour TTL per room.

### `GET /api/v1/rooms/:id/chat/messages?limit=50&after=<iso-timestamp>`

Returns up to `limit` messages, optionally only those after the given
timestamp.

### `POST /api/v1/rooms/:id/chat/messages`

```json
{ "content": "Hello world", "display_name": "Sam" }
```

### `DELETE /api/v1/rooms/:id/chat/messages/:messageId`

Soft-deletes a message. Only the sender can delete their own messages.

## User profile (authenticated)

### `GET /api/v1/users/me`

Returns the current user's profile.

### `PUT /api/v1/users/me`

```json
{ "display_name": "New name", "avatar_url": "https://..." }
```

### `GET /api/v1/users/me/settings`

Returns device defaults, theme, and UI preferences.

### `PUT /api/v1/users/me/settings`

```json
{
  "default_camera_device_id": "abc",
  "default_microphone_device_id": "def",
  "preferred_video_quality": "720p",
  "preferred_theme": "dark",
  "mute_mic_on_join": false,
  "disable_camera_on_join": false,
  "language_code": "en"
}
```

## Persistent rooms (authenticated)

### `GET /api/v1/rooms`

List the current user's rooms.

### `PUT /api/v1/rooms/:id`

Update title, description, scheduled times, or settings.

### `DELETE /api/v1/rooms/:id`

Soft-delete the room.

### `POST /api/v1/rooms/:id/invite`

```json
{ "label": "team", "expires_at": "2026-12-31T00:00:00Z", "max_uses": 10 }
```

Creates a shareable invite link with optional expiry and max usage count.

## Stats (authenticated)

### `GET /api/v1/rooms/:id/stats`

Aggregate connection quality stats for a room.

### `GET /api/v1/rooms/:id/stats/participants/:participantId`

Per-participant network telemetry: average RTT, jitter, packet loss,
bitrate, and a 0–100 network score.

## Server-side screen sharing

Distinct from the browser-native LiveKit screen-share track. These
endpoints let the server broadcast its own display (kiosk / demo).

- `POST /screen-share/offer` — SDP offer + answer exchange.
- `POST /screen-share/ice/:id` — ICE candidate from the client.
- `GET  /screen-share/ice/:id` — poll for server-side ICE candidates.
- `POST /screen-share/hangup/:id` — terminate the session.
- `GET  /screen-share/` — the bundled demo HTML page.

## Error responses

| Status | Meaning                                       |
| ------ | --------------------------------------------- |
| 400    | Malformed request or validation failure.      |
| 401    | Missing or invalid auth token.                |
| 403    | Authenticated but not allowed.                |
| 404    | Resource does not exist.                      |
| 409    | State conflict (e.g. duplicate email).        |
| 429    | Rate limit exceeded.                          |
| 500    | Unexpected server error (logged with stack).  |

All non-2xx responses use the same JSON shape:

```json
{ "error": "description" }
```
