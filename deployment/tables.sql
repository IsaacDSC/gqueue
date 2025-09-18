-- enum type_event
DO $$ BEGIN
    CREATE TYPE type_event AS ENUM ('trigger', 'schedule');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Internal Events Table
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    unique_key TEXT NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    repo_url VARCHAR(500),
    team_owner VARCHAR(255),
    type_event type_event NOT NULL,
    state VARCHAR(100) NOT NULL,
    triggers JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_events_name ON events(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_events_service_name ON events(service_name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_events_type_event ON events(type_event) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_events_state ON events(state) WHERE deleted_at IS NULL;