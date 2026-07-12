package db_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"openplays/server/internal/db"
	"openplays/server/internal/testdb"
)

func TestOAuthUpsertsPreserveCustomAvatar(t *testing.T) {
	tests := []struct {
		name   string
		upsert func(context.Context, *db.Queries, string, string) (db.User, error)
	}{
		{
			name: "google",
			upsert: func(ctx context.Context, queries *db.Queries, id, photoURL string) (db.User, error) {
				providerID := "google-" + id
				return queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
					ID: id, Email: id + "@example.com", DisplayName: id,
					PhotoUrl: &photoURL, OauthPhotoUrl: &photoURL, GoogleID: &providerID,
				})
			},
		},
		{
			name: "facebook",
			upsert: func(ctx context.Context, queries *db.Queries, id, photoURL string) (db.User, error) {
				providerID := "facebook-" + id
				return queries.UpsertUserByFacebookID(ctx, db.UpsertUserByFacebookIDParams{
					ID: id, Email: id + "@example.com", DisplayName: id,
					PhotoUrl: &photoURL, OauthPhotoUrl: &photoURL, FacebookID: &providerID,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB := testdb.New(t)
			queries := db.New(sqlDB)
			ctx := context.Background()

			const userID = "avatar-user"
			const firstProviderPhoto = "https://provider.example/first.jpg"
			if _, err := tt.upsert(ctx, queries, userID, firstProviderPhoto); err != nil {
				t.Fatalf("initial upsert: %v", err)
			}

			const customPhoto = "https://images.openplays.app/avatars/avatar-user/custom.jpg"
			const avatarKey = "avatars/avatar-user/custom.jpg"
			if _, err := sqlDB.ExecContext(ctx,
				"UPDATE users SET photo_url = ?, avatar_key = ? WHERE id = ?",
				customPhoto, avatarKey, userID,
			); err != nil {
				t.Fatalf("set custom avatar: %v", err)
			}

			const refreshedProviderPhoto = "https://provider.example/refreshed.jpg"
			user, err := tt.upsert(ctx, queries, userID, refreshedProviderPhoto)
			if err != nil {
				t.Fatalf("refresh upsert: %v", err)
			}
			if user.PhotoUrl == nil || *user.PhotoUrl != customPhoto {
				t.Fatalf("photo_url = %v, want custom avatar %q", user.PhotoUrl, customPhoto)
			}
			if user.OauthPhotoUrl == nil || *user.OauthPhotoUrl != refreshedProviderPhoto {
				t.Fatalf("oauth_photo_url = %v, want refreshed provider photo %q", user.OauthPhotoUrl, refreshedProviderPhoto)
			}
			if user.AvatarKey == nil || *user.AvatarKey != avatarKey {
				t.Fatalf("avatar_key = %v, want %q", user.AvatarKey, avatarKey)
			}
		})
	}
}

func TestOAuthUpsertRefreshesDisplayedPhotoWithoutCustomAvatar(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	providerID := "google-avatar-user"

	upsert := func(photoURL string) db.User {
		t.Helper()
		user, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
			ID: "avatar-user", Email: "avatar@example.com", DisplayName: "Avatar User",
			PhotoUrl: &photoURL, OauthPhotoUrl: &photoURL, GoogleID: &providerID,
		})
		if err != nil {
			t.Fatalf("upsert: %v", err)
		}
		return user
	}

	upsert("https://provider.example/first.jpg")
	const refreshedPhoto = "https://provider.example/refreshed.jpg"
	user := upsert(refreshedPhoto)

	if user.PhotoUrl == nil || *user.PhotoUrl != refreshedPhoto {
		t.Fatalf("photo_url = %v, want %q", user.PhotoUrl, refreshedPhoto)
	}
	if user.OauthPhotoUrl == nil || *user.OauthPhotoUrl != refreshedPhoto {
		t.Fatalf("oauth_photo_url = %v, want %q", user.OauthPhotoUrl, refreshedPhoto)
	}
}

func TestSetAndClearUserAvatarRestoresOAuthPhoto(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	providerID := "google-avatar-restore"
	providerPhoto := "https://provider.example/photo.jpg"
	if _, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID: "avatar-restore", Email: "restore@example.com", DisplayName: "Restore",
		PhotoUrl: &providerPhoto, OauthPhotoUrl: &providerPhoto, GoogleID: &providerID,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	customPhoto := "https://images.example/avatars/avatar-restore/custom.jpg"
	key := "avatars/avatar-restore/custom.jpg"
	custom, err := queries.SetUserAvatar(ctx, db.SetUserAvatarParams{
		ID: "avatar-restore", PhotoUrl: &customPhoto, AvatarKey: &key,
	})
	if err != nil {
		t.Fatalf("set avatar: %v", err)
	}
	if custom.PhotoUrl == nil || *custom.PhotoUrl != customPhoto || custom.AvatarKey == nil || *custom.AvatarKey != key {
		t.Fatalf("custom avatar = %#v", custom)
	}

	restored, err := queries.ClearUserAvatar(ctx, db.ClearUserAvatarParams{
		ID: "avatar-restore", ExpectedAvatarKey: &key,
	})
	if err != nil {
		t.Fatalf("clear avatar: %v", err)
	}
	if restored.PhotoUrl == nil || *restored.PhotoUrl != providerPhoto || restored.AvatarKey != nil {
		t.Fatalf("restored avatar = %#v", restored)
	}
}

func TestAvatarMutationsRejectStaleAvatarKey(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	providerID := "google-avatar-cas"
	providerPhoto := "https://provider.example/photo.jpg"
	if _, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID: "avatar-cas", Email: "cas@example.com", DisplayName: "CAS",
		PhotoUrl: &providerPhoto, OauthPhotoUrl: &providerPhoto, GoogleID: &providerID,
	}); err != nil {
		t.Fatal(err)
	}

	firstURL, firstKey := "https://images.example/first.jpg", "avatars/avatar-cas/first.jpg"
	if _, err := queries.SetUserAvatar(ctx, db.SetUserAvatarParams{
		ID: "avatar-cas", PhotoUrl: &firstURL, AvatarKey: &firstKey,
	}); err != nil {
		t.Fatal(err)
	}

	secondURL, secondKey := "https://images.example/second.jpg", "avatars/avatar-cas/second.jpg"
	if _, err := queries.SetUserAvatar(ctx, db.SetUserAvatarParams{
		ID: "avatar-cas", PhotoUrl: &secondURL, AvatarKey: &secondKey,
		ExpectedAvatarKey: nil,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("stale set error = %v, want sql.ErrNoRows", err)
	}
	staleKey := "avatars/avatar-cas/stale.jpg"
	if _, err := queries.ClearUserAvatar(ctx, db.ClearUserAvatarParams{
		ID: "avatar-cas", ExpectedAvatarKey: &staleKey,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("stale clear error = %v, want sql.ErrNoRows", err)
	}
}

func TestUpdateUserProfilePreservesOwnedPhotoAndContactFields(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	providerID := "google-profile-owned-fields"
	providerPhoto := "https://provider.example/photo.jpg"
	if _, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID: "owned-fields", Email: "owned@example.com", DisplayName: "Before",
		PhotoUrl: &providerPhoto, OauthPhotoUrl: &providerPhoto, GoogleID: &providerID,
	}); err != nil {
		t.Fatal(err)
	}
	customPhoto := "https://images.example/custom.jpg"
	avatarKey := "avatars/owned-fields/custom.jpg"
	contact := `{"telegram":"@owned"}`
	if _, err := sqlDB.ExecContext(ctx,
		"UPDATE users SET photo_url = ?, avatar_key = ?, contact_info = ? WHERE id = ?",
		customPhoto, avatarKey, contact, "owned-fields",
	); err != nil {
		t.Fatal(err)
	}

	updated, err := queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID: "owned-fields", DisplayName: "After",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.PhotoUrl == nil || *updated.PhotoUrl != customPhoto ||
		updated.AvatarKey == nil || *updated.AvatarKey != avatarKey ||
		updated.ContactInfo == nil || *updated.ContactInfo != contact {
		t.Fatalf("owned fields changed: %#v", updated)
	}
}
