package db_test

import (
	"context"
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
