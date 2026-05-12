// Package auth provides authentication, session management, and OAuth token verification.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

const SessionDuration = 30 * 24 * time.Hour // 30 days rolling

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrAccountBanned = errors.New("account is banned or suspended")
	ErrNoSession     = errors.New("session not found or expired")
)

// Provider identifies an OAuth provider.
type Provider string

const (
	ProviderGoogle   Provider = "google"
	ProviderFacebook Provider = "facebook"
)

// Identity is the provider-agnostic result of verifying an OAuth token.
// Each provider (Google, Facebook) produces one of these.
type Identity struct {
	Provider    Provider
	ProviderID  string // Google sub, Facebook user ID
	Email       string
	DisplayName string
	PhotoURL    string
}

// Store is the database boundary for auth operations.
type Store interface {
	UpsertUserByGoogleID(ctx context.Context, arg db.UpsertUserByGoogleIDParams) (db.User, error)
	UpsertUserByFacebookID(ctx context.Context, arg db.UpsertUserByFacebookIDParams) (db.User, error)
	LinkGoogleID(ctx context.Context, arg db.LinkGoogleIDParams) (db.User, error)
	LinkFacebookID(ctx context.Context, arg db.LinkFacebookIDParams) (db.User, error)
	GetSessionWithUser(ctx context.Context, token string) (db.GetSessionWithUserRow, error)
	CreateSession(ctx context.Context, arg db.CreateSessionParams) error
	DeleteSession(ctx context.Context, token string) error
	RefreshSession(ctx context.Context, arg db.RefreshSessionParams) error
}

// User is the public representation of a user.
type User struct {
	ID            string               `json:"id"`
	Email         string               `json:"email"`
	Username      *string              `json:"username,omitempty"`
	DisplayName   string               `json:"display_name"`
	PhotoURL      *string              `json:"photo_url,omitempty"`
	Status        string               `json:"status"`
	SportsProfile *model.SportsProfile `json:"sports_profile,omitempty"`
	ContactInfo   *string              `json:"contact_info,omitempty"`
	CreatedAt     string               `json:"created_at"`
	UpdatedAt     string               `json:"updated_at"`
}

// LoginResult is returned by Login on success.
type LoginResult struct {
	User         User
	SessionToken string
}

// Service orchestrates auth: user upsert and session management.
// Provider-specific token verification happens outside this service —
// callers verify tokens to produce an Identity, then call Login.
type Service struct {
	store Store
}

// NewService creates an auth service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Login upserts a user from a verified identity and creates a session.
// This is provider-agnostic — Google, Facebook, etc. all use this after
// verifying their respective tokens.
func (s *Service) Login(ctx context.Context, id Identity) (*LoginResult, error) {
	user, err := s.upsertUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("%w: %s", ErrAccountBanned, user.Status)
	}

	token, err := newSessionToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(SessionDuration)
	if err := s.store.CreateSession(ctx, db.CreateSessionParams{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	slog.Info("auth: user logged in",
		"user_id", user.ID,
		"email", user.Email,
		"provider", string(id.Provider),
	)

	return &LoginResult{
		User:         MapUser(user),
		SessionToken: token,
	}, nil
}

// upsertUser dispatches to the correct provider-specific upsert.
// If the email already exists under a different provider, links the new provider to the existing account.
func (s *Service) upsertUser(ctx context.Context, id Identity) (db.User, error) {
	photoURL := ptrOrNil(id.PhotoURL)
	email := normalizeEmail(id.Email)

	var user db.User
	var err error

	switch id.Provider {
	case ProviderGoogle:
		providerID := id.ProviderID
		user, err = s.store.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
			ID:          uuid.New().String(),
			Email:       email,
			DisplayName: id.DisplayName,
			PhotoUrl:    photoURL,
			GoogleID:    &providerID,
		})
		// Email conflict → user exists via Facebook, link Google to that account
		if err != nil && isEmailConflict(err) {
			user, err = s.store.LinkGoogleID(ctx, db.LinkGoogleIDParams{
				GoogleID: &providerID,
				Email:    email,
			})
		}
	case ProviderFacebook:
		providerID := id.ProviderID
		user, err = s.store.UpsertUserByFacebookID(ctx, db.UpsertUserByFacebookIDParams{
			ID:          uuid.New().String(),
			Email:       email,
			DisplayName: id.DisplayName,
			PhotoUrl:    photoURL,
			FacebookID:  &providerID,
		})
		// Email conflict → user exists via Google, link Facebook to that account
		if err != nil && isEmailConflict(err) {
			user, err = s.store.LinkFacebookID(ctx, db.LinkFacebookIDParams{
				FacebookID: &providerID,
				Email:      email,
			})
		}
	default:
		return db.User{}, fmt.Errorf("unsupported provider: %s", id.Provider)
	}

	return user, err
}

func isEmailConflict(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed: users.email")
}

// GetSession validates a session token and returns the user.
// Refreshes the session expiry on each call (rolling 30 days).
func (s *Service) GetSession(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, ErrNoSession
	}

	row, err := s.store.GetSessionWithUser(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	if row.Status != "active" {
		return nil, fmt.Errorf("%w: %s", ErrAccountBanned, row.Status)
	}

	// Rolling refresh
	_ = s.store.RefreshSession(ctx, db.RefreshSessionParams{
		ExpiresAt: time.Now().Add(SessionDuration),
		Token:     token,
	})

	user := mapSessionUser(row)
	return &user, nil
}

// Logout deletes a session.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, token)
}

func MapUser(u db.User) User {
	sportsProfile, _ := model.ParseSportsProfile(u.SportsProfile)
	return User{
		ID:            u.ID,
		Email:         u.Email,
		Username:      u.Username,
		DisplayName:   u.DisplayName,
		PhotoURL:      u.PhotoUrl,
		Status:        u.Status,
		SportsProfile: sportsProfile,
		ContactInfo:   u.ContactInfo,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     u.UpdatedAt.Format(time.RFC3339),
	}
}

func mapSessionUser(s db.GetSessionWithUserRow) User {
	sportsProfile, _ := model.ParseSportsProfile(s.SportsProfile)
	return User{
		ID:            s.UserID2,
		Email:         s.Email,
		Username:      s.Username,
		DisplayName:   s.DisplayName,
		PhotoURL:      s.PhotoUrl,
		Status:        s.Status,
		SportsProfile: sportsProfile,
		ContactInfo:   s.ContactInfo,
		CreatedAt:     s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     s.UpdatedAt.Format(time.RFC3339),
	}
}

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// normalizeEmail strips plus addressing (user+tag@domain → user@domain)
// and lowercases the entire address.
func normalizeEmail(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at == -1 {
		return email
	}
	local := email[:at]
	domain := email[at:]
	if plus := strings.Index(local, "+"); plus != -1 {
		local = local[:plus]
	}
	return local + domain
}
