CREATE TABLE note_shares (
    id TEXT PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    note_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    access_role TEXT NOT NULL CHECK (access_role IN ('read_only', 'full_access')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active')),
    invited_by_user_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(note_id, email)
);

CREATE INDEX idx_note_shares_note_id ON note_shares (note_id);
CREATE INDEX idx_note_shares_email ON note_shares (email);
