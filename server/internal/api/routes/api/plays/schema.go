package plays

import (
	"fmt"
	"time"

	"openplays/server/internal/model"
)

// PlayPublic is the API response schema for a play.
type PlayPublic struct {
	ID          string            `json:"id"`
	CreatedAt   *string           `json:"created_at,omitempty"`
	UpdatedAt   *string           `json:"updated_at,omitempty"`
	ListingType model.ListingType `json:"listing_type"`
	Sport       model.Sport       `json:"sport"`
	GameType    *model.GameType   `json:"game_type,omitempty"`
	HostName    string            `json:"host_name"`
	StartsAt    string            `json:"starts_at"`
	EndsAt      string            `json:"ends_at"`
	Timezone    string            `json:"timezone"`
	Venue       string            `json:"venue" doc:"Raw venue name as extracted from the message"`
	VenueName   string            `json:"venue_name" doc:"Display name: resolved venue name, or raw venue name, or 'No venue'"`
	VenueID     *int64            `json:"venue_id,omitempty"`

	VenuePostalCode *string  `json:"venue_postal_code,omitempty"`
	VenueLatitude   *float64 `json:"venue_latitude,omitempty"`
	VenueLongitude  *float64 `json:"venue_longitude,omitempty"`

	LevelMin *string `json:"level_min,omitempty"`
	LevelMax *string `json:"level_max,omitempty"`

	Fee        *int64 `json:"fee,omitempty"` // cents
	Currency   string `json:"currency"`
	MaxPlayers *int64 `json:"max_players,omitempty"`
	SlotsLeft  *int64 `json:"slots_left,omitempty"`
	Courts     *int64 `json:"courts,omitempty"`

	Contacts   model.Contacts    `json:"contacts"`
	GenderPref *model.GenderPref `json:"gender_pref,omitempty"`
	Meta       model.Meta        `json:"meta"`

	Source           *string `json:"source,omitempty"`
	SourceSenderLink *string `json:"source_sender_link,omitempty" doc:"Link to sender's Telegram profile, e.g. t.me/username"`
	SourceMessageID  *string `json:"source_message_id,omitempty"`
	SourceGroup      *string `json:"source_group,omitempty"`
	SourceLink       *string `json:"source_link,omitempty" doc:"Deep link to original message, e.g. t.me/group/123"`

	// Creator info (null for telegram-scraped plays)
	CreatedBy          *string `json:"created_by,omitempty"`
	CreatorDisplayName *string `json:"creator_display_name,omitempty"`
	CreatorUsername    *string `json:"creator_username,omitempty"`
	CreatorPhotoURL    *string `json:"creator_photo_url,omitempty"`

	ParticipantPreview []PlayParticipantPreviewPublic `json:"participant_preview,omitempty"`

	ConfirmedParticipants []PlayParticipantPreviewPublic `json:"confirmed_participants,omitempty"`
	Waitlist              []PlayParticipantPreviewPublic `json:"waitlist,omitempty"`
	ViewerState           *string                        `json:"viewer_state,omitempty" enum:"not_joined,confirmed,waitlisted,creator"`
	CanManage             *bool                          `json:"can_manage,omitempty"`
	ConfirmedCount        *int64                         `json:"confirmed_count,omitempty"`
	WaitlistCount         *int64                         `json:"waitlist_count,omitempty"`

	distanceKm float64 `json:"-"`
}

func publicPlayTimestamps(createdBy *string, createdAt, updatedAt time.Time) (*string, *string) {
	if createdBy != nil {
		return nil, nil
	}
	created := createdAt.Format(time.RFC3339)
	updated := updatedAt.Format(time.RFC3339)
	return &created, &updated
}

// PlayParticipantPreviewPublic is the compact roster data shown on play cards.
type PlayParticipantPreviewPublic struct {
	ID          int64   `json:"id"`
	UserID      *string `json:"-"`
	DisplayName *string `json:"display_name,omitempty"`
	PhotoURL    *string `json:"photo_url,omitempty"`
	RatingCode  *string `json:"rating_code,omitempty"`
	IsGuest     bool    `json:"is_guest"`
	IsHost      bool    `json:"is_host"`
}

// buildSourceLink constructs a deep link to the original message.
// Returns nil if the source is not supported or fields are missing.
func buildSourceLink(source, group, messageID *string) *string {
	if source == nil || group == nil || messageID == nil {
		return nil
	}
	if *source == "telegram" {
		link := fmt.Sprintf("https://t.me/%s/%s", *group, *messageID)
		return &link
	}
	return nil
}

// buildSenderLink constructs a link to the sender's Telegram profile.
// Returns nil if the username is not available.
func buildSenderLink(source, username *string) *string {
	if source == nil || username == nil || *username == "" {
		return nil
	}
	if *source == "telegram" {
		link := fmt.Sprintf("https://t.me/%s", *username)
		return &link
	}
	return nil
}
