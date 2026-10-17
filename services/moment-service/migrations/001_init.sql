-- moment-service schema: moments, cuts, jobs, idempotency_keys
-- Apply with: psql "$DATABASE_URL" -f migrations/001_init.sql

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS moments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             TEXT NOT NULL,
    status              TEXT NOT NULL CHECK (status IN ('intake', 'drafting', 'rendering', 'ready', 'failed')),
    song_pick_id        TEXT NOT NULL,
    asset_ids           UUID[] NOT NULL DEFAULT '{}',
    cut_id              UUID,
    render_storage_key  TEXT,
    error_code          TEXT,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS moments_user_id_idx ON moments (user_id);

CREATE TABLE IF NOT EXISTS cuts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    moment_id       UUID NOT NULL UNIQUE REFERENCES moments (id) ON DELETE CASCADE,
    version         INT NOT NULL DEFAULT 1,
    song_pick_id    TEXT NOT NULL,
    clips           JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE moments
    DROP CONSTRAINT IF EXISTS moments_cut_id_fkey;
ALTER TABLE moments
    ADD CONSTRAINT moments_cut_id_fkey
    FOREIGN KEY (cut_id) REFERENCES cuts (id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    moment_id       UUID NOT NULL REFERENCES moments (id) ON DELETE CASCADE,
    job_type        TEXT NOT NULL CHECK (job_type IN ('cut-draft', 'render')),
    status          TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'superseded')),
    attempt         INT NOT NULL DEFAULT 0,
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    result          JSONB,
    error_code      TEXT,
    error_message   TEXT,
    cut_version     INT,
    idempotency_key TEXT,
    claimed_by      TEXT,
    claimed_at      TIMESTAMPTZ,
    lease_until     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS jobs_claim_idx ON jobs (job_type, status, created_at);
CREATE INDEX IF NOT EXISTS jobs_moment_id_idx ON jobs (moment_id);

-- At most one active (queued|running) job per (moment_id, job_type).
CREATE UNIQUE INDEX IF NOT EXISTS jobs_one_active_per_type_idx
    ON jobs (moment_id, job_type)
    WHERE status IN ('queued', 'running');

CREATE TABLE IF NOT EXISTS idempotency_keys (
    user_id         TEXT NOT NULL,
    key             TEXT NOT NULL,
    request_hash    TEXT NOT NULL,
    moment_id       UUID NOT NULL REFERENCES moments (id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, key)
);
