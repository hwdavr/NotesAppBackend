ALTER TABLE note_block_comments
    ADD COLUMN parent_comment_id TEXT NULL REFERENCES note_block_comments(id) ON DELETE SET NULL,
    ADD COLUMN mentions JSONB NOT NULL DEFAULT '[]'::jsonb;
