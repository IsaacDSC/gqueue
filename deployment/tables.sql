-- Internal Events Table
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    unique_key TEXT NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    repo_url VARCHAR(500),
    team_owner VARCHAR(255),
    triggers JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_internal_events_name ON internal_events(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_internal_events_service_name ON internal_events(service_name) WHERE deleted_at IS NULL;