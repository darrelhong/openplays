package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/pagination"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/sqliteutils"
	"openplays/server/internal/usernames"
)

type ShoutoutsInput struct {
	Username string `path:"username" doc:"Username"`
	Cursor   string `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit    int64  `query:"limit" default:"10" minimum:"1" maximum:"50" doc:"Number of results per page"`
}

// PublicUserShoutout is an attributed piece of praise on a profile. It is
// the only review payload that carries reviewer identity, and it never
// carries a rating.
type PublicUserShoutout struct {
	Shoutout            string      `json:"shoutout"`
	CreatedAt           string      `json:"created_at"`
	ReviewerDisplayName string      `json:"reviewer_display_name"`
	ReviewerUsername    *string     `json:"reviewer_username,omitempty"`
	ReviewerPhotoURL    *string     `json:"reviewer_photo_url,omitempty"`
	PlayID              string      `json:"play_id"`
	Sport               model.Sport `json:"sport"`
	PlayName            *string     `json:"play_name,omitempty"`
	PlayStartsAt        string      `json:"play_starts_at"`
	PlayTimezone        string      `json:"play_timezone"`
}

type ShoutoutsOutput struct {
	Body pagination.Page[PublicUserShoutout]
}

// ShoutoutsStore is the DB boundary for a profile's shoutout list.
type ShoutoutsStore interface {
	GetActiveUserProfileByUsername(ctx context.Context, username *string) (db.GetActiveUserProfileByUsernameRow, error)
	ListUserShoutouts(ctx context.Context, arg db.ListUserShoutoutsParams) ([]db.ListUserShoutoutsRow, error)
	CountUserShoutouts(ctx context.Context, revieweeUserID string) (int64, error)
}

func RegisterShoutouts(api huma.API, store ShoutoutsStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-user-shoutouts",
		Summary:     "List a user's shoutouts",
		Description: "Returns a page of attributed shoutouts on a user's profile, newest first. Requires authentication.",
		Method:      http.MethodGet,
		Path:        "/{username}/shoutouts",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *ShoutoutsInput) (*ShoutoutsOutput, error) {
		username, err := usernames.Normalize(input.Username)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}
		row, err := store.GetActiveUserProfileByUsername(ctx, &username)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("user not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get user")
		}

		var cursorCreatedAt interface{}
		var cursorID *int64
		if createdAt, id, ok := decodeShoutoutCursor(input.Cursor); ok {
			cursorCreatedAt = createdAt.UTC().Format(sqliteutils.DateTimeFormat)
			cursorID = &id
		}

		rows, err := store.ListUserShoutouts(ctx, db.ListUserShoutoutsParams{
			RevieweeUserID:  row.ID,
			CursorCreatedAt: cursorCreatedAt,
			CursorID:        cursorID,
			PageSize:        input.Limit + 1,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list shoutouts")
		}
		total, err := store.CountUserShoutouts(ctx, row.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count shoutouts")
		}

		page := pagination.Paginate(rows, input.Limit, total, func(r db.ListUserShoutoutsRow) string {
			return encodeShoutoutCursor(r.CreatedAt, r.ID)
		})

		out := &ShoutoutsOutput{}
		out.Body = pagination.Page[PublicUserShoutout]{
			Total:      page.Total,
			NextCursor: page.NextCursor,
			HasMore:    page.HasMore,
			Items:      make([]PublicUserShoutout, 0, len(page.Items)),
		}
		for _, item := range page.Items {
			out.Body.Items = append(out.Body.Items, publicUserShoutout(item))
		}
		return out, nil
	})
}

func publicUserShoutout(row db.ListUserShoutoutsRow) PublicUserShoutout {
	shoutout := ""
	if row.Shoutout != nil {
		shoutout = *row.Shoutout
	}
	return PublicUserShoutout{
		Shoutout:            shoutout,
		CreatedAt:           row.CreatedAt.Format(time.RFC3339),
		ReviewerDisplayName: row.ReviewerDisplayName,
		ReviewerUsername:    row.ReviewerUsername,
		ReviewerPhotoURL:    row.ReviewerPhotoUrl,
		PlayID:              row.PlayID,
		Sport:               row.Sport,
		PlayName:            row.PlayName,
		PlayStartsAt:        row.StartsAt.Format(time.RFC3339),
		PlayTimezone:        row.Timezone,
	}
}

// --- (created_at, review id) cursor ---

func encodeShoutoutCursor(createdAt time.Time, id int64) string {
	return fmt.Sprintf("%s,%d", createdAt.UTC().Format(time.RFC3339), id)
}

func decodeShoutoutCursor(cursor string) (createdAt time.Time, id int64, ok bool) {
	if cursor == "" {
		return time.Time{}, 0, false
	}
	parts := strings.SplitN(cursor, ",", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, false
	}
	createdAt, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, 0, false
	}
	id, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, false
	}
	return createdAt, id, true
}
