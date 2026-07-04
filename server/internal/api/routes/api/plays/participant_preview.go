package plays

import (
	"context"
	"fmt"
	"strings"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

func hydrateParticipantPreviews(ctx context.Context, queries ParticipantPreviewBatchStore, items []PlayPublic, includeNames bool) error {
	if len(items) == 0 {
		return nil
	}

	playIDs := make([]string, 0, len(items))
	for i := range items {
		playIDs = append(playIDs, items[i].ID)
	}

	rows, err := queries.ListRosteredParticipantPreviewsByPlays(ctx, playIDs)
	if err != nil {
		return fmt.Errorf("list participant previews: %w", err)
	}

	rowsByPlayID := make(map[string][]participantPreviewRow, len(items))
	for _, row := range rows {
		rowsByPlayID[row.PlayID] = append(rowsByPlayID[row.PlayID], batchParticipantPreviewRow(row))
	}

	hostUserIDsByPlay, err := hostUserIDListsByPlay(ctx, queries, playIDs)
	if err != nil {
		return fmt.Errorf("list play hosts: %w", err)
	}

	for i := range items {
		hostIDs := hostUserIDsByPlay[items[i].ID]
		previews := mapParticipantPreviewRows(items[i].Sport, rowsByPlayID[items[i].ID], includeNames)
		markHostParticipants(previews, userIDSet(hostIDs))
		previews, err = appendMissingHostPreviews(ctx, queries, items[i].ID, items[i].Sport, includeNames, previews, hostIDs)
		if err != nil {
			return fmt.Errorf("append missing host previews: %w", err)
		}
		items[i].ParticipantPreview = orderHostPreviewsFirst(previews, hostIDs)
	}
	return nil
}

// playRosterPreviews is the play's full roster partitioned by participant
// status, with host and viewer rows marked.
type playRosterPreviews struct {
	Confirmed  []PlayParticipantPreviewPublic
	Added      []PlayParticipantPreviewPublic
	Requested  []PlayParticipantPreviewPublic
	Waitlisted []PlayParticipantPreviewPublic
}

// rosterPreviewsForPlay fetches every participant of a play in one query and
// partitions the previews by status. Hosts without a roster row are appended
// to the confirmed list (and listed first), matching the roster the UI shows.
func rosterPreviewsForPlay(ctx context.Context, queries *db.Queries, playID string, sport model.Sport, includeNames bool, hostIDs []string, viewerID *string) (playRosterPreviews, error) {
	rows, err := queries.ListParticipantPreviewsByPlay(ctx, playID)
	if err != nil {
		return playRosterPreviews{}, err
	}

	rowsByStatus := make(map[model.PlayParticipantStatus][]participantPreviewRow, 4)
	for _, row := range rows {
		rowsByStatus[row.Status] = append(rowsByStatus[row.Status], playParticipantPreviewRow(row))
	}

	hostIDSet := userIDSet(hostIDs)
	partition := func(status model.PlayParticipantStatus) []PlayParticipantPreviewPublic {
		previews := mapParticipantPreviewRows(sport, rowsByStatus[status], includeNames)
		markHostParticipants(previews, hostIDSet)
		if viewerID != nil {
			markViewerParticipant(previews, *viewerID)
		}
		return previews
	}

	roster := playRosterPreviews{
		Confirmed:  partition(model.ParticipantConfirmed),
		Added:      partition(model.ParticipantAdded),
		Requested:  partition(model.ParticipantRequested),
		Waitlisted: partition(model.ParticipantWaitlisted),
	}

	confirmed, err := appendMissingHostPreviews(ctx, queries, playID, sport, includeNames, roster.Confirmed, hostIDs)
	if err != nil {
		return playRosterPreviews{}, err
	}
	roster.Confirmed = orderHostPreviewsFirst(confirmed, hostIDs)
	return roster, nil
}

type participantPreviewRow struct {
	ID            int64
	PlayID        string
	UserID        *string
	GuestName     *string
	RatingCode    *string
	DisplayName   *string
	Username      *string
	PhotoUrl      *string
	SportsProfile *string
}

func playParticipantPreviewRow(row db.ListParticipantPreviewsByPlayRow) participantPreviewRow {
	return participantPreviewRow{
		ID:            row.ID,
		PlayID:        row.PlayID,
		UserID:        row.UserID,
		GuestName:     row.GuestName,
		RatingCode:    row.RatingCode,
		DisplayName:   row.DisplayName,
		Username:      row.Username,
		PhotoUrl:      row.PhotoUrl,
		SportsProfile: row.SportsProfile,
	}
}

func batchParticipantPreviewRow(row db.ListRosteredParticipantPreviewsByPlaysRow) participantPreviewRow {
	return participantPreviewRow{
		ID:            row.ID,
		PlayID:        row.PlayID,
		UserID:        row.UserID,
		GuestName:     row.GuestName,
		RatingCode:    row.RatingCode,
		DisplayName:   row.DisplayName,
		Username:      row.Username,
		PhotoUrl:      row.PhotoUrl,
		SportsProfile: row.SportsProfile,
	}
}

func mapParticipantPreviewRows(sport model.Sport, rows []participantPreviewRow, includeNames bool) []PlayParticipantPreviewPublic {
	previews := make([]PlayParticipantPreviewPublic, 0, len(rows))
	for _, row := range rows {
		previews = append(previews, PlayParticipantPreviewPublic{
			ID:          row.ID,
			UserID:      row.UserID,
			DisplayName: participantPreviewName(row, includeNames),
			Username:    row.Username,
			PhotoURL:    cleanStringPtr(row.PhotoUrl),
			RatingCode:  participantPreviewRating(sport, row),
			IsGuest:     row.UserID == nil,
		})
	}
	return previews
}

func appendMissingHostPreviews(ctx context.Context, queries interface {
	GetUserByID(context.Context, string) (db.User, error)
}, playID string, sport model.Sport, includeNames bool, previews []PlayParticipantPreviewPublic, hostIDs []string) ([]PlayParticipantPreviewPublic, error) {
	if len(hostIDs) == 0 {
		return previews, nil
	}

	seenUserIDs := make(map[string]struct{}, len(previews))
	for _, preview := range previews {
		if preview.UserID != nil {
			seenUserIDs[*preview.UserID] = struct{}{}
		}
	}

	for index, hostID := range hostIDs {
		if _, ok := seenUserIDs[hostID]; ok {
			continue
		}

		user, err := queries.GetUserByID(ctx, hostID)
		if err != nil {
			return nil, err
		}

		id := hostID
		displayName := user.DisplayName
		rows := []participantPreviewRow{{
			ID:            -int64(index + 1),
			PlayID:        playID,
			UserID:        &id,
			DisplayName:   &displayName,
			Username:      user.Username,
			PhotoUrl:      user.PhotoUrl,
			SportsProfile: user.SportsProfile,
		}}
		hostPreview := mapParticipantPreviewRows(sport, rows, includeNames)[0]
		hostPreview.IsHost = true
		previews = append(previews, hostPreview)
	}

	return previews, nil
}

func orderHostPreviewsFirst(previews []PlayParticipantPreviewPublic, hostIDs []string) []PlayParticipantPreviewPublic {
	if len(previews) < 2 || len(hostIDs) == 0 {
		return previews
	}

	ordered := make([]PlayParticipantPreviewPublic, 0, len(previews))
	used := make([]bool, len(previews))
	for _, hostID := range hostIDs {
		for i, preview := range previews {
			if used[i] || preview.UserID == nil || *preview.UserID != hostID {
				continue
			}
			ordered = append(ordered, preview)
			used[i] = true
		}
	}
	if len(ordered) == 0 {
		return previews
	}

	for i, preview := range previews {
		if !used[i] {
			ordered = append(ordered, preview)
		}
	}
	return ordered
}

func hostUserIDListsByPlay(ctx context.Context, queries ParticipantPreviewBatchStore, playIDs []string) (map[string][]string, error) {
	rows, err := queries.ListPlayHostUserIDsByPlays(ctx, playIDs)
	if err != nil {
		return nil, err
	}

	out := make(map[string][]string, len(playIDs))
	for _, row := range rows {
		out[row.PlayID] = append(out[row.PlayID], row.UserID)
	}
	return out, nil
}

func userIDSet(ids []string) map[string]struct{} {
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func markHostParticipants(previews []PlayParticipantPreviewPublic, hostUserIDs map[string]struct{}) {
	if len(hostUserIDs) == 0 {
		return
	}
	for i := range previews {
		if previews[i].UserID == nil {
			continue
		}
		if _, ok := hostUserIDs[*previews[i].UserID]; ok {
			previews[i].IsHost = true
		}
	}
}

func participantPreviewName(row participantPreviewRow, includeNames bool) *string {
	if !includeNames {
		return nil
	}
	if row.UserID == nil {
		return cleanStringPtr(row.GuestName)
	}
	return cleanStringPtr(row.DisplayName)
}

func participantPreviewRating(sport model.Sport, row participantPreviewRow) *string {
	if rating := cleanStringPtr(row.RatingCode); rating != nil {
		return rating
	}

	profile, err := model.ParseSportsProfile(row.SportsProfile)
	if err != nil {
		return nil
	}
	return profile.LevelFor(sport)
}

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
