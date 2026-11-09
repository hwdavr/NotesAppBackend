CREATE TABLE notion_link_users (
    user_id TEXT PRIMARY KEY,
    generation BIGINT NOT NULL DEFAULT 0,
    account_name TEXT,
    encrypted_token BYTEA
);
CREATE TABLE notion_oauth_states (
    state_hash BYTEA PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES notion_link_users(user_id) ON DELETE CASCADE,
    generation BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX notion_oauth_states_user_id_idx ON notion_oauth_states(user_id);
