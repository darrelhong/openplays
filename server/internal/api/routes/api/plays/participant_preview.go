package plays

import (
	"context"
	"fmt"
	"strings"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

func hydrateParticipantPreviews(ctx context.Context, queries *db.Queries, items []PlayPublic, includeNames bool) error {
	if len(items) == 0 {
		return nil
	}

	playIDs := make([]string, 0, len(items))
	for i := range items {
		playIDs = append(playIDs, items[i].ID)
	}

	rows, err := queries.ListConfirmedParticipantPreviewsByPlays(ctx, playIDs)
	if err != nil {
		return fmt.Errorf("list participant previews: %w", err)
	}

	rowsByPlayID := make(map[string][]participantPreviewRow, len(items))
	for _, row := range rows {
		rowsByPlayID[row.PlayID] = append(rowsByPlayID[row.PlayID], batchParticipantPreviewRow(row))
	}

	hostUserIDsByPlay, err := hostUserIDSetsByPlay(ctx, queries, playIDs)
	if err != nil {
		return fmt.Errorf("list play hosts: %w", err)
	}

	for i := range items {
		previews := mapParticipantPreviewRows(items[i].Sport, rowsByPlayID[items[i].ID], includeNames)
		markHostParticipants(previews, hostUserIDsByPlay[items[i].ID])
		items[i].ParticipantPreview = previews
	}
	return nil
}

func participantPreviewsForPlay(ctx context.Context, queries *db.Queries, playID string, sport model.Sport, includeNames bool) ([]PlayParticipantPreviewPublic, error) {
	rows, err := queries.ListConfirmedParticipantPreviewsByPlay(ctx, playID)
	if err != nil {
		return nil, err
	}
	previews := mapParticipantPreviewRows(sport, singleParticipantPreviewRows(rows), includeNames)
	hostUserIDs, err := hostUserIDSetForPlay(ctx, queries, playID)
	if err != nil {
		return nil, err
	}
	markHostParticipants(previews, hostUserIDs)
	return previews, nil
}

func participantPreviewsForPlayByStatus(ctx context.Context, queries *db.Queries, playID string, sport model.Sport, status model.PlayParticipantStatus, includeNames bool) ([]PlayParticipantPreviewPublic, error) {
	rows, err := queries.ListParticipantPreviewsByPlayAndStatus(ctx, db.ListParticipantPreviewsByPlayAndStatusParams{
		PlayID: playID,
		Status: status,
	})
	if err != nil {
		return nil, err
	}
	previews := mapParticipantPreviewRows(sport, statusParticipantPreviewRows(rows), includeNames)
	hostUserIDs, err := hostUserIDSetForPlay(ctx, queries, playID)
	if err != nil {
		return nil, err
	}
	markHostParticipants(previews, hostUserIDs)
	return previews, nil
}

type participantPreviewRow struct {
	ID            int64
	PlayID        string
	UserID        *string
	GuestName     *string
	RatingCode    *string
	DisplayName   *string
	PhotoUrl      *string
	SportsProfile *string
}

func singleParticipantPreviewRows(rows []db.ListConfirmedParticipantPreviewsByPlayRow) []participantPreviewRow {
	out := make([]participantPreviewRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, participantPreviewRow{
			ID:            row.ID,
			PlayID:        row.PlayID,
			UserID:        row.UserID,
			GuestName:     row.GuestName,
			RatingCode:    row.RatingCode,
			DisplayName:   row.DisplayName,
			PhotoUrl:      row.PhotoUrl,
			SportsProfile: row.SportsProfile,
		})
	}
	return out
}

func batchParticipantPreviewRow(row db.ListConfirmedParticipantPreviewsByPlaysRow) participantPreviewRow {
	return participantPreviewRow{
		ID:            row.ID,
		PlayID:        row.PlayID,
		UserID:        row.UserID,
		GuestName:     row.GuestName,
		RatingCode:    row.RatingCode,
		DisplayName:   row.DisplayName,
		PhotoUrl:      row.PhotoUrl,
		SportsProfile: row.SportsProfile,
	}
}

func statusParticipantPreviewRows(rows []db.ListParticipantPreviewsByPlayAndStatusRow) []participantPreviewRow {
	out := make([]participantPreviewRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, participantPreviewRow{
			ID:            row.ID,
			PlayID:        row.PlayID,
			UserID:        row.UserID,
			GuestName:     row.GuestName,
			RatingCode:    row.RatingCode,
			DisplayName:   row.DisplayName,
			PhotoUrl:      row.PhotoUrl,
			SportsProfile: row.SportsProfile,
		})
	}
	return out
}

func mapParticipantPreviewRows(sport model.Sport, rows []participantPreviewRow, includeNames bool) []PlayParticipantPreviewPublic {
	previews := make([]PlayParticipantPreviewPublic, 0, len(rows))
	for _, row := range rows {
		previews = append(previews, PlayParticipantPreviewPublic{
			ID:          row.ID,
			UserID:      row.UserID,
			DisplayName: participantPreviewName(row, includeNames),
			PhotoURL:    cleanStringPtr(row.PhotoUrl),
			RatingCode:  participantPreviewRating(sport, row),
			IsGuest:     row.UserID == nil,
		})
	}
	return previews
}

func hostUserIDSetForPlay(ctx context.Context, queries *db.Queries, playID string) (map[string]struct{}, error) {
	ids, err := queries.ListPlayHostUserIDsByPlay(ctx, playID)
	if err != nil {
		return nil, err
	}
	return userIDSet(ids), nil
}

func hostUserIDSetsByPlay(ctx context.Context, queries *db.Queries, playIDs []string) (map[string]map[string]struct{}, error) {
	rows, err := queries.ListPlayHostUserIDsByPlays(ctx, playIDs)
	if err != nil {
		return nil, err
	}

	out := make(map[string]map[string]struct{}, len(playIDs))
	for _, row := range rows {
		if out[row.PlayID] == nil {
			out[row.PlayID] = make(map[string]struct{})
		}
		out[row.PlayID][row.UserID] = struct{}{}
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
