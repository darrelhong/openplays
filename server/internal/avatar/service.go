package avatar

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	"openplays/server/internal/db"
	"openplays/server/internal/objectstore"
)

const avatarCacheControl = "public, max-age=31536000, immutable"

type ObjectStore interface {
	Put(context.Context, string, io.Reader, objectstore.PutOptions) error
	Delete(context.Context, string) error
	PublicURL(string) (string, error)
}

type UserStore interface {
	GetUserByID(context.Context, string) (db.User, error)
	SetUserAvatar(context.Context, db.SetUserAvatarParams) (db.User, error)
	ClearUserAvatar(context.Context, db.ClearUserAvatarParams) (db.User, error)
}

type ImageProcessor interface {
	Process(io.Reader) (ProcessedImage, error)
}

type Service struct {
	objects   ObjectStore
	users     UserStore
	processor ImageProcessor
}

func NewService(objects ObjectStore, users UserStore, processor ImageProcessor) *Service {
	return &Service{objects: objects, users: users, processor: processor}
}

func (s *Service) Upload(ctx context.Context, userID string, input io.Reader) (db.User, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return db.User{}, fmt.Errorf("get avatar user: %w", err)
	}
	processed, err := s.processor.Process(input)
	if err != nil {
		return db.User{}, err
	}

	key := fmt.Sprintf("avatars/%s/%s%s", userID, uuid.NewString(), processed.Extension)
	publicURL, err := s.objects.PublicURL(key)
	if err != nil {
		return db.User{}, fmt.Errorf("build avatar URL: %w", err)
	}
	if err := s.objects.Put(ctx, key, bytes.NewReader(processed.Data), objectstore.PutOptions{
		ContentType: processed.ContentType, CacheControl: avatarCacheControl,
	}); err != nil {
		return db.User{}, fmt.Errorf("store avatar: %w", err)
	}

	updated, err := s.users.SetUserAvatar(ctx, db.SetUserAvatarParams{
		PhotoUrl: &publicURL, AvatarKey: &key, ID: userID,
		ExpectedAvatarKey: user.AvatarKey,
	})
	if err != nil {
		// An UPDATE ... RETURNING can commit even if Scan reports a canceled
		// context. Re-read before deleting so we never remove a referenced object.
		current, checkErr := s.users.GetUserByID(context.WithoutCancel(ctx), userID)
		if checkErr == nil && current.AvatarKey != nil && *current.AvatarKey == key {
			return current, nil
		}
		if checkErr == nil {
			s.deleteBestEffort(context.WithoutCancel(ctx), key, "uncommitted")
		}
		return db.User{}, fmt.Errorf("save avatar: %w", err)
	}

	if user.AvatarKey != nil && *user.AvatarKey != key {
		s.deleteBestEffort(context.WithoutCancel(ctx), *user.AvatarKey, "replaced")
	}
	return updated, nil
}

func (s *Service) Remove(ctx context.Context, userID string) (db.User, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return db.User{}, fmt.Errorf("get avatar user: %w", err)
	}
	if user.AvatarKey == nil {
		return user, nil
	}

	updated, err := s.users.ClearUserAvatar(ctx, db.ClearUserAvatarParams{
		ID: userID, ExpectedAvatarKey: user.AvatarKey,
	})
	if err != nil {
		current, checkErr := s.users.GetUserByID(context.WithoutCancel(ctx), userID)
		if checkErr == nil && current.AvatarKey == nil {
			s.deleteBestEffort(context.WithoutCancel(ctx), *user.AvatarKey, "removed")
			return current, nil
		}
		return db.User{}, fmt.Errorf("clear avatar: %w", err)
	}
	s.deleteBestEffort(context.WithoutCancel(ctx), *user.AvatarKey, "removed")
	return updated, nil
}

func (s *Service) deleteBestEffort(ctx context.Context, key, reason string) {
	if err := s.objects.Delete(ctx, key); err != nil {
		slog.Warn("avatar: failed to delete "+reason+" object", "key", key, "error", err)
	}
}
