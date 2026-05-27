CREATE TABLE note_block_comments (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    note_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    block_id TEXT NOT NULL,
    author_user_id TEXT NOT NULL,
    author_display_name TEXT NULL,
    author_email TEXT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_note_block_comments_note_block ON note_block_comments (note_id, block_id);
CREATE INDEX idx_note_block_comments_author ON note_block_comments (author_user_id);
