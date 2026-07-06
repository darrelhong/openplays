package users

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/usernames"
)

type ProfileInput struct {
	Username string `path:"username" doc:"Username"`
}

type PublicUserProfile struct {
	ID                string                   `json:"id"`
	DisplayName       string                   `json:"display_name"`
	Username          string                   `json:"username"`
	PhotoURL          *string                  `json:"photo_url,omitempty"`
	SportsProfile     *model.SportsProfile     `json:"sports_profile,omitempty"`
	Sports            []PublicUserProfileSport `json:"sports"`
	RosteredPlayCount int64                    `json:"rostered_play_count"`
	// Review reputation. The rating is an anonymous aggregate; props count
	// under the sport they were earned in; shoutouts are attributed and are
	// the only place reviewer identity appears.
	Rating    *PublicUserRating     `json:"rating,omitempty"`
	Props     []PublicUserPropCount `json:"props"`
	Shoutouts []PublicUserShoutout  `json:"shoutouts"`
}

type PublicUserProfileSport struct {
	Sport             model.Sport `json:"sport"`
	RatingCode        *string     `json:"rating_code,omitempty"`
	RosteredPlayCount int64       `json:"rostered_play_count"`
}

type PublicUserRating struct {
	Average float64 `json:"average"`
	Count   int64   `json:"count"`
	// Distribution holds the count of 1- through 5-star ratings, in that
	// order, for the profile's rating breakdown chart.
	Distribution [5]int64 `json:"distribution"`
}

type PublicUserPropCount struct {
	Sport model.Sport `json:"sport"`
	Prop  string      `json:"prop"`
	Count int64       `json:"count"`
}

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

// shoutoutLimit caps the profile's shoutout list; newest first.
const shoutoutLimit = 20

type ProfileOutput struct {
	Body PublicUserProfile
}

func RegisterProfile(api huma.API, store ProfileStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-user-profile",
		Summary:     "Get user profile",
		Description: "Returns a minimal public profile for an active user. Requires authentication.",
		Method:      http.MethodGet,
		Path:        "/{username}",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *ProfileInput) (*ProfileOutput, error) {
		username, err := usernames.Normalize(input.Username)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}

		row, err := store.GetActiveUserProfileByUsername(ctx, &username)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("user not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get user profile")
		}
		count, err := store.CountRosteredPlaysByUser(ctx, row.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count user plays")
		}
		sportCounts, err := store.CountRosteredPlaysByUserAndSport(ctx, row.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count user plays by sport")
		}

		profile := mapPublicUserProfile(row, count, sportCounts)
		if err := hydrateReviewReputation(ctx, store, &profile); err != nil {
			return nil, huma.Error500InternalServerError("failed to get user reviews")
		}

		return &ProfileOutput{Body: profile}, nil
	})
}

func hydrateReviewReputation(ctx context.Context, store ProfileStore, profile *PublicUserProfile) error {
	rating, err := store.GetUserRatingAggregate(ctx, profile.ID)
	if err != nil {
		return err
	}
	if rating.RatingCount > 0 {
		profile.Rating = &PublicUserRating{Average: rating.Average, Count: rating.RatingCount}
		buckets, err := store.ListUserRatingDistribution(ctx, profile.ID)
		if err != nil {
			return err
		}
		for _, bucket := range buckets {
			if bucket.Rating != nil && *bucket.Rating >= 1 && *bucket.Rating <= 5 {
				profile.Rating.Distribution[*bucket.Rating-1] = bucket.RatingCount
			}
		}
	}

	propCounts, err := store.ListUserPropCounts(ctx, profile.ID)
	if err != nil {
		return err
	}
	profile.Props = make([]PublicUserPropCount, 0, len(propCounts))
	for _, row := range propCounts {
		profile.Props = append(profile.Props, PublicUserPropCount{
			Sport: row.Sport,
			Prop:  row.Prop,
			Count: row.PropCount,
		})
	}

	shoutouts, err := store.ListUserShoutouts(ctx, db.ListUserShoutoutsParams{
		RevieweeUserID: profile.ID,
		Limit:          shoutoutLimit,
	})
	if err != nil {
		return err
	}
	profile.Shoutouts = make([]PublicUserShoutout, 0, len(shoutouts))
	for _, row := range shoutouts {
		if row.Shoutout == nil {
			continue
		}
		profile.Shoutouts = append(profile.Shoutouts, PublicUserShoutout{
			Shoutout:            *row.Shoutout,
			CreatedAt:           row.CreatedAt.Format(time.RFC3339),
			ReviewerDisplayName: row.ReviewerDisplayName,
			ReviewerUsername:    row.ReviewerUsername,
			ReviewerPhotoURL:    row.ReviewerPhotoUrl,
			PlayID:              row.PlayID,
			Sport:               row.Sport,
			PlayName:            row.PlayName,
			PlayStartsAt:        row.StartsAt.Format(time.RFC3339),
			PlayTimezone:        row.Timezone,
		})
	}
	return nil
}

func mapPublicUserProfile(row db.GetActiveUserProfileByUsernameRow, rosteredPlayCount int64, sportCounts []db.CountRosteredPlaysByUserAndSportRow) PublicUserProfile {
	profile, _ := model.ParseSportsProfile(row.SportsProfile)
	username := ""
	if row.Username != nil {
		username = *row.Username
	}
	return PublicUserProfile{
		ID:                row.ID,
		DisplayName:       row.DisplayName,
		Username:          username,
		PhotoURL:          row.PhotoUrl,
		SportsProfile:     profile,
		Sports:            publicUserProfileSports(profile, sportCounts),
		RosteredPlayCount: rosteredPlayCount,
	}
}

func publicUserProfileSports(profile *model.SportsProfile, sportCounts []db.CountRosteredPlaysByUserAndSportRow) []PublicUserProfileSport {
	countsBySport := make(map[model.Sport]int64, len(sportCounts))
	for _, row := range sportCounts {
		countsBySport[row.Sport] = row.PlayCount
	}

	out := make([]PublicUserProfileSport, 0, len(model.SportValues))
	for _, value := range model.SportValues {
		sport := model.Sport(value)
		ratingCode := profile.LevelFor(sport)
		count := countsBySport[sport]
		if ratingCode == nil && count == 0 {
			continue
		}
		out = append(out, PublicUserProfileSport{
			Sport:             sport,
			RatingCode:        ratingCode,
			RosteredPlayCount: count,
		})
	}
	return out
}
