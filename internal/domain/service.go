package domain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hwdavr/notes-app-backend/internal/pkg/email"
)

var (
	ErrInvalidItem  = errors.New("invalid item payload")
	ErrItemNotFound = errors.New("item not found")
	ErrSyncConflict = errors.New("sync conflict")
	ErrInvalidMove  = errors.New("invalid move")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
)

type Service struct {
	Repo  *Repository
	Email email.Service
}

func NewService(r *Repository, e email.Service) *Service {
	return &Service{Repo: r, Email: e}
}

func (s *Service) CreateFolder(ctx context.Context, userID, userEmail string, input CreateItemInput) (Item, error) {
	input.Type = ItemTypeFolder
	input.Name = strings.TrimSpace(input.Name)
	input.Content = ""
	input.SortKey = strings.TrimSpace(input.SortKey)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	if userID == "" || input.Name == "" || input.SortKey == "" || input.DeviceID == "" {
		return Item{}, ErrInvalidItem
	}
	if err := s.validateParent(ctx, userID, userEmail, "", input.ParentID); err != nil {
		return Item{}, err
	}
	return s.Repo.CreateItem(ctx, userID, input)
}

func (s *Service) CreateNote(ctx context.Context, userID, userEmail string, input CreateItemInput) (Item, error) {
	input.Type = ItemTypeNote
	input.Name = strings.TrimSpace(input.Name)
	input.SortKey = strings.TrimSpace(input.SortKey)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	if userID == "" || input.Name == "" || input.SortKey == "" || input.DeviceID == "" {
		return Item{}, ErrInvalidItem
	}
	if err := s.validateParent(ctx, userID, userEmail, "", input.ParentID); err != nil {
		return Item{}, err
	}
	return s.Repo.CreateItem(ctx, userID, input)
}

func (s *Service) ListItems(ctx context.Context, userID, userEmail string, filter ListItemsFilter) ([]Item, error) {
	return s.Repo.ListItems(ctx, userID, userEmail, filter)
}

func (s *Service) GetItem(ctx context.Context, userID, userEmail, itemID string) (Item, error) {
	return s.Repo.GetItem(ctx, userID, userEmail, itemID)
}

func (s *Service) RenameItem(ctx context.Context, userID, userEmail, itemID, name, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	name = strings.TrimSpace(name)
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || name == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, userEmail, itemID)
	if err != nil {
		return MutationResult{}, err
	}

	if current.AccessRole != AccessRoleFullAccess {
		return MutationResult{}, ErrUnauthorized
	}

	update := UpdateItemInput{
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	}
	if lastSyncedVersion >= current.Version || current.Name == name {
		update.Name = &name
	}

	item, err := s.Repo.UpdateItem(ctx, userID, userEmail, itemID, update)
	if err != nil {
		return MutationResult{}, err
	}
	if update.Name == nil {
		return MutationResult{
			Status:         "conflict",
			Item:           item,
			ConflictFields: []string{"name"},
			Message:        "server kept newer value for: name",
		}, ErrSyncConflict
	}

	return MutationResult{Status: "merged", Item: item}, nil
}

func (s *Service) UpdateNoteContent(ctx context.Context, userID, userEmail, itemID, content, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, userEmail, itemID)
	if err != nil {
		return MutationResult{}, err
	}
	if current.Type != ItemTypeNote {
		return MutationResult{}, ErrInvalidItem
	}

	if current.AccessRole != AccessRoleFullAccess {
		return MutationResult{}, ErrUnauthorized
	}

	update := UpdateItemInput{
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	}
	if lastSyncedVersion >= current.Version || current.Content == content {
		update.Content = &content
	}

	item, err := s.Repo.UpdateItem(ctx, userID, userEmail, itemID, update)
	if err != nil {
		return MutationResult{}, err
	}
	if update.Content == nil {
		return MutationResult{
			Status:         "conflict",
			Item:           item,
			ConflictFields: []string{"content"},
			Message:        "server kept newer value for: content",
		}, ErrSyncConflict
	}

	return MutationResult{Status: "merged", Item: item}, nil
}

func (s *Service) MoveItem(ctx context.Context, userID, userEmail, itemID, deviceID string, parentID *string, lastSyncedVersion int64) (MutationResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, userEmail, itemID)
	if err != nil {
		return MutationResult{}, err
	}

	if parentID != nil {
		trimmed := strings.TrimSpace(*parentID)
		if trimmed == "" {
			parentID = nil
		} else {
			parentID = &trimmed
		}
	}
	if err := s.validateParent(ctx, userID, userEmail, itemID, parentID); err != nil {
		return MutationResult{}, err
	}

	if current.Type == ItemTypeFolder && parentID != nil {
		hasDescendant, err := s.Repo.FolderHasDescendant(ctx, userID, itemID, *parentID)
		if err != nil {
			return MutationResult{}, err
		}
		if hasDescendant {
			return MutationResult{}, ErrInvalidMove
		}
	}

	if current.AccessRole != AccessRoleFullAccess {
		return MutationResult{}, ErrUnauthorized
	}

	item, err := s.Repo.UpdateItem(ctx, userID, userEmail, itemID, UpdateItemInput{
		ParentID:          parentID,
		ClearParentID:     parentID == nil,
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	})
	if err != nil {
		return MutationResult{}, err
	}

	return MutationResult{Status: "merged", Item: item}, nil
}

func (s *Service) validateParent(ctx context.Context, userID, userEmail, itemID string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	if *parentID == "" || *parentID == itemID {
		return ErrInvalidMove
	}

	parent, err := s.Repo.GetItem(ctx, userID, userEmail, *parentID)
	if err != nil {
		return err
	}
	if parent.Type != ItemTypeFolder || parent.DeletedAt != nil {
		return ErrInvalidMove
	}
	return nil
}

func (s *Service) ReorderItem(ctx context.Context, userID, userEmail, itemID, sortKey, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	sortKey = strings.TrimSpace(sortKey)
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || sortKey == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, userEmail, itemID)
	if err != nil {
		return MutationResult{}, err
	}
	if current.AccessRole != AccessRoleFullAccess {
		return MutationResult{}, ErrUnauthorized
	}

	item, err := s.Repo.UpdateItem(ctx, userID, userEmail, itemID, UpdateItemInput{
		SortKey:           &sortKey,
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	})
	if err != nil {
		return MutationResult{}, err
	}

	return MutationResult{Status: "merged", Item: item}, nil
}

func (s *Service) DeleteItem(ctx context.Context, userID, userEmail, itemID, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	deletedAt := time.Now().UTC()
	if err := s.Repo.SoftDeleteItemTree(ctx, userID, itemID, deviceID, lastSyncedVersion, deletedAt); err != nil {
		return MutationResult{}, err
	}

	item, err := s.Repo.GetItem(ctx, userID, userEmail, itemID)
	if err != nil {
		return MutationResult{}, err
	}

	return MutationResult{Status: "merged", Item: item}, nil
}

func (s *Service) FavoriteItem(ctx context.Context, userID, userEmail, itemID, deviceID string, isFavorite bool, lastSyncedVersion int64) (MutationResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, userEmail, itemID)
	if err != nil {
		return MutationResult{}, err
	}

	if current.AccessRole != AccessRoleFullAccess {
		return MutationResult{}, ErrUnauthorized
	}

	update := UpdateItemInput{
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	}
	if lastSyncedVersion >= current.Version || current.IsFavorite == isFavorite {
		update.IsFavorite = &isFavorite
	}

	item, err := s.Repo.UpdateItem(ctx, userID, userEmail, itemID, update)
	if err != nil {
		return MutationResult{}, err
	}
	if update.IsFavorite == nil {
		return MutationResult{
			Status:         "conflict",
			Item:           item,
			ConflictFields: []string{"isFavorite"},
			Message:        "server kept newer value for: isFavorite",
		}, ErrSyncConflict
	}

	return MutationResult{Status: "merged", Item: item}, nil
}

func (s *Service) ListNoteShares(ctx context.Context, userID, noteID string) ([]NoteShare, error) {
	// Verify user has access to the note
	_, err := s.Repo.GetItem(ctx, userID, "", noteID)
	if err != nil {
		return nil, err
	}

	return s.Repo.ListNoteShares(ctx, noteID)
}

func (s *Service) CreateNoteShare(ctx context.Context, userID, noteID string, input CreateNoteShareRequest) (NoteShare, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Email == "" || (input.AccessRole != AccessRoleReadOnly && input.AccessRole != AccessRoleFullAccess) {
		return NoteShare{}, ErrInvalidItem
	}

	// Verify user is the owner of the note
	note, err := s.Repo.GetItem(ctx, userID, "", noteID)
	if err != nil {
		return NoteShare{}, err
	}
	if note.UserID != userID {
		return NoteShare{}, ErrItemNotFound // Or Unauthorized if we had that error
	}

	// Check if already shared
	_, err = s.Repo.GetNoteShareByNoteAndEmail(ctx, noteID, input.Email)
	if err == nil {
		// Already shared
		return NoteShare{}, ErrConflict
	}

	share, err := s.Repo.CreateNoteShare(ctx, noteID, input.Email, input.AccessRole, ShareStatusActive, userID)
	if err != nil {
		return NoteShare{}, err
	}

	// Send email
	if err := s.Email.SendInvite(input.Email, note.Name, userID); err != nil {
		// Log error but don't fail the request?
		// For now, let's just log it.
	}

	return share, nil
}

func (s *Service) UpdateNoteShare(ctx context.Context, userID, noteID, shareID string, input UpdateNoteShareRequest) (NoteShare, error) {
	if input.AccessRole != AccessRoleReadOnly && input.AccessRole != AccessRoleFullAccess {
		return NoteShare{}, ErrInvalidItem
	}

	// Verify user is the owner of the note
	note, err := s.Repo.GetItem(ctx, userID, "", noteID)
	if err != nil {
		return NoteShare{}, err
	}
	if note.UserID != userID {
		return NoteShare{}, ErrItemNotFound
	}

	return s.Repo.UpdateNoteShare(ctx, noteID, shareID, input.AccessRole)
}

func (s *Service) DeleteNoteShare(ctx context.Context, userID, noteID, shareID string) error {
	// Verify user is the owner of the note
	note, err := s.Repo.GetItem(ctx, userID, "", noteID)
	if err != nil {
		return err
	}
	if note.UserID != userID {
		return ErrItemNotFound
	}

	return s.Repo.DeleteNoteShare(ctx, noteID, shareID)
}
