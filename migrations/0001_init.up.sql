CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE items (
  id TEXT PRIMARY KEY DEFAULT uuid_generate_v4()::text,
  user_id TEXT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('folder', 'note')),
  parent_id TEXT NULL,
  name TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  sort_key TEXT NOT NULL,
  version BIGINT NOT NULL DEFAULT 1,
  device_id TEXT NOT NULL,
  last_synced_version BIGINT NOT NULL DEFAULT 0,
  deleted_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_items_user_parent_sort ON items (user_id, parent_id, sort_key);
CREATE INDEX idx_items_user_updated_at ON items (user_id, updated_at DESC);
CREATE INDEX idx_items_user_version ON items (user_id, version DESC);
CREATE INDEX idx_items_user_type ON items (user_id, type);
