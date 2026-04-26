package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidItem  = errors.New("invalid item payload")
	ErrItemNotFound = errors.New("item not found")
	ErrSyncConflict = errors.New("sync conflict")
	ErrInvalidMove  = errors.New("invalid move")
)

type Service struct {
	Repo *Repository
}

func NewService(r *Repository) *Service {
	return &Service{Repo: r}
}

func (s *Service) CreateFolder(ctx context.Context, userID string, input CreateItemInput) (Item, error) {
	input.Type = ItemTypeFolder
	input.Name = strings.TrimSpace(input.Name)
	input.Content = ""
	input.SortKey = strings.TrimSpace(input.SortKey)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	if userID == "" || input.Name == "" || input.SortKey == "" || input.DeviceID == "" {
		return Item{}, ErrInvalidItem
	}
	if err := s.validateParent(ctx, userID, "", input.ParentID); err != nil {
		return Item{}, err
	}
	return s.Repo.CreateItem(ctx, userID, input)
}

func (s *Service) CreateNote(ctx context.Context, userID string, input CreateItemInput) (Item, error) {
	input.Type = ItemTypeNote
	input.Name = strings.TrimSpace(input.Name)
	input.SortKey = strings.TrimSpace(input.SortKey)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	if userID == "" || input.Name == "" || input.SortKey == "" || input.DeviceID == "" {
		return Item{}, ErrInvalidItem
	}
	if err := s.validateParent(ctx, userID, "", input.ParentID); err != nil {
		return Item{}, err
	}
	return s.Repo.CreateItem(ctx, userID, input)
}

func (s *Service) ListItems(ctx context.Context, userID string, filter ListItemsFilter) ([]Item, error) {
	return s.Repo.ListItems(ctx, userID, filter)
}

func (s *Service) GetItem(ctx context.Context, userID, itemID string) (Item, error) {
	return s.Repo.GetItem(ctx, userID, itemID)
}

func (s *Service) RenameItem(ctx context.Context, userID, itemID, name, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	name = strings.TrimSpace(name)
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || name == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, itemID)
	if err != nil {
		return MutationResult{}, err
	}

	update := UpdateItemInput{
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	}
	if lastSyncedVersion >= current.Version || current.Name == name {
		update.Name = &name
	}

	item, err := s.Repo.UpdateItem(ctx, userID, itemID, update)
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

func (s *Service) UpdateNoteContent(ctx context.Context, userID, itemID, content, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, itemID)
	if err != nil {
		return MutationResult{}, err
	}
	if current.Type != ItemTypeNote {
		return MutationResult{}, ErrInvalidItem
	}

	update := UpdateItemInput{
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	}
	if lastSyncedVersion >= current.Version || current.Content == content {
		update.Content = &content
	}

	item, err := s.Repo.UpdateItem(ctx, userID, itemID, update)
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

func (s *Service) MoveItem(ctx context.Context, userID, itemID, deviceID string, parentID *string, lastSyncedVersion int64) (MutationResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	current, err := s.Repo.GetItem(ctx, userID, itemID)
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
	if err := s.validateParent(ctx, userID, itemID, parentID); err != nil {
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

	item, err := s.Repo.UpdateItem(ctx, userID, itemID, UpdateItemInput{
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

func (s *Service) validateParent(ctx context.Context, userID, itemID string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	if *parentID == "" || *parentID == itemID {
		return ErrInvalidMove
	}

	parent, err := s.Repo.GetItem(ctx, userID, *parentID)
	if err != nil {
		return err
	}
	if parent.Type != ItemTypeFolder || parent.DeletedAt != nil {
		return ErrInvalidMove
	}
	return nil
}

func (s *Service) ReorderItem(ctx context.Context, userID, itemID, sortKey, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	sortKey = strings.TrimSpace(sortKey)
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || sortKey == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	item, err := s.Repo.UpdateItem(ctx, userID, itemID, UpdateItemInput{
		SortKey:           &sortKey,
		DeviceID:          deviceID,
		LastSyncedVersion: lastSyncedVersion,
	})
	if err != nil {
		return MutationResult{}, err
	}

	return MutationResult{Status: "merged", Item: item}, nil
}

func (s *Service) DeleteItem(ctx context.Context, userID, itemID, deviceID string, lastSyncedVersion int64) (MutationResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == "" || itemID == "" || deviceID == "" {
		return MutationResult{}, ErrInvalidItem
	}

	deletedAt := time.Now().UTC()
	if err := s.Repo.SoftDeleteItemTree(ctx, userID, itemID, deviceID, lastSyncedVersion, deletedAt); err != nil {
		return MutationResult{}, err
	}

	item, err := s.Repo.GetItem(ctx, userID, itemID)
	if err != nil {
		return MutationResult{}, err
	}

	return MutationResult{Status: "merged", Item: item}, nil
}
