ALTER TABLE note_block_comments
    DROP COLUMN IF EXISTS mentions,
    DROP COLUMN IF EXISTS parent_comment_id;
