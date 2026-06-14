package plays

import (
	"context"

	"openplays/server/internal/db"
)

// CreatePlayStore is the DB boundary for creating plays.
type CreatePlayStore interface {
	CreatePlay(ctx context.Context, arg db.CreatePlayParams) (db.Play, error)
	CreatePlayHost(ctx context.Context, arg db.CreatePlayHostParams) (db.PlayHost, error)
	CreatePlayParticipant(ctx context.Context, arg db.CreatePlayParticipantParams) (db.PlayParticipant, error)
}

// ParticipantPreviewBatchStore is the DB boundary for hydrating roster previews on play lists.
type ParticipantPreviewBatchStore interface {
	ListConfirmedParticipantPreviewsByPlays(ctx context.Context, playIds []string) ([]db.ListConfirmedParticipantPreviewsByPlaysRow, error)
	ListPlayHostUserIDsByPlays(ctx context.Context, playIds []string) ([]db.ListPlayHostUserIDsByPlaysRow, error)
	GetUserByID(ctx context.Context, id string) (db.User, error)
}

// ViewerStateBatchStore is the DB boundary for hydrating the current user's
// relationship to play list items.
type ViewerStateBatchStore interface {
	ListPlayHostUserIDsByPlays(ctx context.Context, playIds []string) ([]db.ListPlayHostUserIDsByPlaysRow, error)
	ListPlayParticipantStatesByUserAndPlays(ctx context.Context, arg db.ListPlayParticipantStatesByUserAndPlaysParams) ([]db.ListPlayParticipantStatesByUserAndPlaysRow, error)
}


// MyPlayStore is the DB boundary for the current user's private play list.
type MyPlayStore interface {
	ParticipantPreviewBatchStore
	ListMyUpcomingPlays(ctx context.Context, arg db.ListMyUpcomingPlaysParams) ([]db.ListMyUpcomingPlaysRow, error)
	CountMyUpcomingPlays(ctx context.Context, userID *string) (int64, error)
}
