package plays

import (
	"context"

	"openplays/server/internal/db"
)

// CreatePlayStore is the DB boundary for creating plays.
type CreatePlayStore interface {
	CreatePlay(ctx context.Context, arg db.CreatePlayParams) (db.Play, error)
	CreatePlayParticipant(ctx context.Context, arg db.CreatePlayParticipantParams) (db.PlayParticipant, error)
}
