package domain

import "time"

const (
	ItemTypeFolder = "folder"
	ItemTypeNote   = "note"
)

type Item struct {
	ID                string     `db:"id" json:"id"`
	UserID            string     `db:"user_id" json:"userId"`
	Type              string     `db:"type" json:"type"`
	ParentID          *string    `db:"parent_id" json:"parentId"`
	Name              string     `db:"name" json:"name"`
	Content           string     `db:"content" json:"content"`
	SortKey           string     `db:"sort_key" json:"sortKey"`
	Version           int64      `db:"version" json:"version"`
	DeviceID          string     `db:"device_id" json:"deviceId"`
	LastSyncedVersion int64      `db:"last_synced_version" json:"lastSyncedVersion"`
	DeletedAt         *time.Time `db:"deleted_at" json:"deletedAt"`
	CreatedAt         time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updatedAt"`
	IsFavorite        bool       `db:"is_favorite" json:"isFavorite"`
}

type CreateItemInput struct {
	ID       string
	Type     string
	ParentID *string
	Name     string
	Content  string
	SortKey  string
	DeviceID string
}

type UpdateItemInput struct {
	Name              *string
	Content           *string
	ParentID          *string
	ClearParentID     bool
	SortKey           *string
	DeletedAt         *time.Time
	ClearDeletedAt    bool
	DeviceID          string
	LastSyncedVersion int64
	IsFavorite        *bool
}

type ListItemsFilter struct {
	Type           string
	ParentID       *string
	RootOnly       bool
	Query          string
	SinceVersion   *int64
	IncludeDeleted bool
}

type MutationResult struct {
	Status         string   `json:"status"`
	Item           Item     `json:"item"`
	ConflictFields []string `json:"conflictFields,omitempty"`
	Message        string   `json:"message,omitempty"`
}

const (
	AccessRoleReadOnly   = "read_only"
	AccessRoleFullAccess = "full_access"
)

const (
	ShareStatusPending = "pending"
	ShareStatusActive  = "active"
)

type NoteShare struct {
	ID              string    `db:"id" json:"id"`
	NoteID          string    `db:"note_id" json:"noteId"`
	Email           string    `db:"email" json:"email"`
	AccessRole      string    `db:"access_role" json:"accessRole"`
	Status          string    `db:"status" json:"status"`
	InvitedByUserID string    `db:"invited_by_user_id" json:"invitedByUserId"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
}

type CreateNoteShareRequest struct {
	Email      string `json:"email"`
	AccessRole string `json:"accessRole"`
}

type UpdateNoteShareRequest struct {
	AccessRole string `json:"accessRole"`
}
