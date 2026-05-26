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

	for i := range items {
		items[i].ParticipantPreview = mapParticipantPreviewRows(items[i].Sport, rowsByPlayID[items[i].ID], includeNames)
	}
	return nil
}

func participantPreviewsForPlay(ctx context.Context, queries *db.Queries, playID string, sport model.Sport, includeNames bool) ([]PlayParticipantPreviewPublic, error) {
	rows, err := queries.ListConfirmedParticipantPreviewsByPlay(ctx, playID)
	if err != nil {
		return nil, err
	}
	return mapParticipantPreviewRows(sport, singleParticipantPreviewRows(rows), includeNames), nil
}

func participantPreviewsForPlayByStatus(ctx context.Context, queries *db.Queries, playID string, sport model.Sport, status model.PlayParticipantStatus, includeNames bool) ([]PlayParticipantPreviewPublic, error) {
	rows, err := queries.ListParticipantPreviewsByPlayAndStatus(ctx, db.ListParticipantPreviewsByPlayAndStatusParams{
		PlayID: playID,
		Status: status,
	})
	if err != nil {
		return nil, err
	}
	return mapParticipantPreviewRows(sport, statusParticipantPreviewRows(rows), includeNames), nil
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
			DisplayName: participantPreviewName(row, includeNames),
			PhotoURL:    cleanStringPtr(row.PhotoUrl),
			RatingCode:  participantPreviewRating(sport, row),
			IsGuest:     row.UserID == nil,
		})
	}
	return previews
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
