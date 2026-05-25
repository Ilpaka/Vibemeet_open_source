-- =============================================================================
--  Vibemeet — database schema
-- =============================================================================
--  This script is executed once on first container start
--  (docker-entrypoint-initdb.d). It is idempotent: every object is created
--  with IF NOT EXISTS, so re-running it against an existing database is safe.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- -----------------------------------------------------------------------------
--  Users and sessions
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS users (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username            VARCHAR(50),
    email               CITEXT UNIQUE NOT NULL,
    password_hash       TEXT NOT NULL,
    display_name        TEXT NOT NULL,
    avatar_url          TEXT,
    global_role         TEXT NOT NULL DEFAULT 'user'
                        CHECK (global_role IN ('user', 'technical_admin')),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    is_email_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email       ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username    ON users(username) WHERE username IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_created_at  ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_global_role ON users(global_role);

-- Refresh token store. Access tokens are stateless JWTs; refresh tokens are
-- persisted (hashed) so that they can be revoked.
CREATE TABLE IF NOT EXISTS user_sessions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT NOT NULL UNIQUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,
    ip_address          INET,
    user_agent          TEXT
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id     ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at  ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token_hash  ON user_sessions(refresh_token_hash);

CREATE TABLE IF NOT EXISTS user_settings (
    user_id                       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    default_camera_device_id      TEXT,
    default_microphone_device_id  TEXT,
    default_speaker_device_id     TEXT,
    preferred_video_quality       TEXT NOT NULL DEFAULT 'auto'
                                  CHECK (preferred_video_quality IN ('1080p','720p','480p','360p','auto')),
    preferred_theme               TEXT NOT NULL DEFAULT 'system'
                                  CHECK (preferred_theme IN ('light','dark','system')),
    mute_mic_on_join              BOOLEAN NOT NULL DEFAULT FALSE,
    disable_camera_on_join        BOOLEAN NOT NULL DEFAULT FALSE,
    language_code                 VARCHAR(8) DEFAULT 'en',
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- -----------------------------------------------------------------------------
--  Authenticated rooms (host owns the room, can invite, lock, etc.)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS rooms (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    livekit_room_name     TEXT UNIQUE NOT NULL,
    host_user_id          UUID NOT NULL REFERENCES users(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    status                TEXT NOT NULL DEFAULT 'scheduled'
                          CHECK (status IN ('scheduled','active','ended','cancelled')),
    scheduled_start_at    TIMESTAMPTZ,
    scheduled_end_at      TIMESTAMPTZ,
    actual_start_at       TIMESTAMPTZ,
    actual_end_at         TIMESTAMPTZ,
    max_participants      INTEGER NOT NULL DEFAULT 10
                          CHECK (max_participants > 0 AND max_participants <= 500),
    waiting_room_enabled  BOOLEAN NOT NULL DEFAULT TRUE,
    is_locked             BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash         TEXT,
    settings              JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rooms_host              ON rooms(host_user_id);
CREATE INDEX IF NOT EXISTS idx_rooms_status            ON rooms(status);
CREATE INDEX IF NOT EXISTS idx_rooms_scheduled_start   ON rooms(scheduled_start_at);
CREATE INDEX IF NOT EXISTS idx_rooms_livekit_name      ON rooms(livekit_room_name);

CREATE TABLE IF NOT EXISTS room_invites (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id             UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    created_by_user_id  UUID NOT NULL REFERENCES users(id),
    link_token          TEXT NOT NULL UNIQUE,
    label               TEXT,
    expires_at          TIMESTAMPTZ,
    max_uses            INTEGER,
    used_count          INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_room_invites_room_id  ON room_invites(room_id);
CREATE INDEX IF NOT EXISTS idx_room_invites_token    ON room_invites(link_token);
CREATE INDEX IF NOT EXISTS idx_room_invites_expires  ON room_invites(expires_at);

CREATE TABLE IF NOT EXISTS room_participants (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id         UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id),
    role            TEXT NOT NULL CHECK (role IN ('host','co_host','participant')),
    display_name    TEXT NOT NULL,
    livekit_sid     TEXT UNIQUE,
    joined_at       TIMESTAMPTZ NOT NULL,
    left_at         TIMESTAMPTZ,
    leave_reason    TEXT,
    is_kicked       BOOLEAN NOT NULL DEFAULT FALSE,
    initial_muted   BOOLEAN NOT NULL DEFAULT FALSE,
    client_ip       INET,
    user_agent      TEXT
);

CREATE INDEX IF NOT EXISTS idx_rp_room_id_joined_at  ON room_participants(room_id, joined_at);
CREATE INDEX IF NOT EXISTS idx_rp_user_id            ON room_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_rp_livekit_sid        ON room_participants(livekit_sid);
CREATE INDEX IF NOT EXISTS idx_rp_left_at            ON room_participants(left_at) WHERE left_at IS NULL;

-- Pre-join approval queue for moderated rooms.
CREATE TABLE IF NOT EXISTS waiting_room_entries (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id             UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id             UUID REFERENCES users(id),
    display_name        TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','approved','rejected','expired')),
    requested_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at          TIMESTAMPTZ,
    decided_by_user_id  UUID REFERENCES users(id),
    reason              TEXT
);

CREATE INDEX IF NOT EXISTS idx_wre_room_status     ON waiting_room_entries(room_id, status);
CREATE INDEX IF NOT EXISTS idx_wre_user            ON waiting_room_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_wre_requested_at    ON waiting_room_entries(requested_at);

-- -----------------------------------------------------------------------------
--  Chat (authenticated rooms persist messages in Postgres;
--  anonymous rooms keep messages in Redis with a 6h TTL — see
--  AnonymousChatRepository)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS chat_messages (
    id                          BIGSERIAL PRIMARY KEY,
    room_id                     UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    sender_participant_id       UUID REFERENCES room_participants(id),
    message_type                TEXT NOT NULL DEFAULT 'user'
                                CHECK (message_type IN ('user','system')),
    content                     TEXT NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at                   TIMESTAMPTZ,
    deleted_at                  TIMESTAMPTZ,
    deleted_by_participant_id   UUID REFERENCES room_participants(id)
);

CREATE INDEX IF NOT EXISTS idx_chat_room_created_at  ON chat_messages(room_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_sender           ON chat_messages(sender_participant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_deleted          ON chat_messages(deleted_at) WHERE deleted_at IS NULL;

-- -----------------------------------------------------------------------------
--  Per-participant connection quality (collected from LiveKit telemetry)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS participant_stats (
    id                   BIGSERIAL PRIMARY KEY,
    room_participant_id  UUID NOT NULL UNIQUE REFERENCES room_participants(id) ON DELETE CASCADE,
    avg_rtt_ms           NUMERIC(10,2),
    max_rtt_ms           NUMERIC(10,2),
    avg_jitter_ms        NUMERIC(10,2),
    packet_loss_up_pct   NUMERIC(5,2),
    packet_loss_down_pct NUMERIC(5,2),
    avg_bitrate_kbps     NUMERIC(10,2),
    network_score        SMALLINT CHECK (network_score >= 0 AND network_score <= 100),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ps_network_score  ON participant_stats(network_score);
CREATE INDEX IF NOT EXISTS idx_ps_created_at     ON participant_stats(created_at);

-- -----------------------------------------------------------------------------
--  Per-user video presets (blur / background / noise suppression)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS user_video_profiles (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                     TEXT NOT NULL,
    background_type          TEXT NOT NULL CHECK (background_type IN ('none','blur','image')),
    background_image_url     TEXT,
    noise_suppression_level  TEXT NOT NULL DEFAULT 'medium'
                             CHECK (noise_suppression_level IN ('off','low','medium','high')),
    is_default               BOOLEAN NOT NULL DEFAULT FALSE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uvp_user     ON user_video_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_uvp_default  ON user_video_profiles(user_id, is_default);

-- -----------------------------------------------------------------------------
--  Audit log
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS audit_log (
    id              BIGSERIAL PRIMARY KEY,
    event_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_user_id   UUID REFERENCES users(id),
    actor_role      TEXT NOT NULL CHECK (actor_role IN ('user','host','technical_admin','system')),
    room_id         UUID REFERENCES rooms(id),
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}'::JSONB
);

CREATE INDEX IF NOT EXISTS idx_audit_room_time    ON audit_log(room_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor_time   ON audit_log(actor_user_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_audit_event_type   ON audit_log(event_type, event_time DESC);

-- -----------------------------------------------------------------------------
--  Anonymous rooms — no auth required. Created on-demand, expire automatically.
--  Used as the default flow in the web client.
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS anonymous_rooms (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    livekit_room_name   VARCHAR(255) UNIQUE NOT NULL,
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    status              VARCHAR(50) NOT NULL DEFAULT 'active',
    max_participants    INTEGER DEFAULT 10,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_anonymous_rooms_status      ON anonymous_rooms(status);
CREATE INDEX IF NOT EXISTS idx_anonymous_rooms_expires_at  ON anonymous_rooms(expires_at);
CREATE INDEX IF NOT EXISTS idx_anonymous_rooms_created_at  ON anonymous_rooms(created_at);

CREATE TABLE IF NOT EXISTS anonymous_participants (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id         UUID NOT NULL REFERENCES anonymous_rooms(id) ON DELETE CASCADE,
    participant_id  VARCHAR(255) NOT NULL,
    display_name    VARCHAR(255) NOT NULL,
    role            VARCHAR(50) NOT NULL DEFAULT 'participant',
    livekit_sid     VARCHAR(255),
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at         TIMESTAMPTZ,
    client_ip       INET,
    user_agent      TEXT,
    CONSTRAINT unique_room_participant UNIQUE (room_id, participant_id, left_at)
);

CREATE INDEX IF NOT EXISTS idx_anonymous_participants_room            ON anonymous_participants(room_id);
CREATE INDEX IF NOT EXISTS idx_anonymous_participants_active          ON anonymous_participants(room_id, left_at);
CREATE INDEX IF NOT EXISTS idx_anonymous_participants_participant_id  ON anonymous_participants(participant_id);

-- -----------------------------------------------------------------------------
--  Triggers
-- -----------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_users_updated_at           ON users;
DROP TRIGGER IF EXISTS update_rooms_updated_at           ON rooms;
DROP TRIGGER IF EXISTS update_user_settings_updated_at   ON user_settings;
DROP TRIGGER IF EXISTS update_anonymous_rooms_updated_at ON anonymous_rooms;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rooms_updated_at
    BEFORE UPDATE ON rooms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_anonymous_rooms_updated_at
    BEFORE UPDATE ON anonymous_rooms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- -----------------------------------------------------------------------------
--  Table comments
-- -----------------------------------------------------------------------------

COMMENT ON TABLE users                  IS 'Registered users (host accounts and admins).';
COMMENT ON TABLE user_sessions          IS 'Refresh tokens (hashed) for revocable session management.';
COMMENT ON TABLE user_settings          IS 'Per-user device defaults and UI preferences.';
COMMENT ON TABLE rooms                  IS 'Persistent conference rooms owned by a host user.';
COMMENT ON TABLE room_invites           IS 'Shareable invite links for persistent rooms.';
COMMENT ON TABLE room_participants      IS 'Participants of persistent rooms with roles and join/leave times.';
COMMENT ON TABLE waiting_room_entries   IS 'Pre-join approval queue for moderated rooms.';
COMMENT ON TABLE chat_messages          IS 'Chat history for persistent rooms.';
COMMENT ON TABLE participant_stats      IS 'Connection quality snapshot per participant session.';
COMMENT ON TABLE user_video_profiles    IS 'Video presets (background, blur, noise suppression).';
COMMENT ON TABLE audit_log              IS 'Append-only log of significant events.';
COMMENT ON TABLE anonymous_rooms        IS 'Ephemeral rooms created without authentication.';
COMMENT ON TABLE anonymous_participants IS 'Participants of anonymous rooms (identified by participant_id cookie).';
