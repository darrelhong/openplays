package plays

import (
	"context"
	"database/sql"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
)

type playHostReader interface {
	GetPlayHost(ctx context.Context, arg db.GetPlayHostParams) (db.PlayHost, error)
}

func isPlayHost(ctx context.Context, store playHostReader, playID, userID string) (bool, error) {
	_, err := store.GetPlayHost(ctx, db.GetPlayHostParams{
		PlayID: playID,
		UserID: userID,
	})
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, huma.Error500InternalServerError("failed to check play host permissions")
	}
	return true, nil
}

func requirePlayHost(ctx context.Context, store playHostReader, playID, userID string) error {
	ok, err := isPlayHost(ctx, store, playID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return huma.Error403Forbidden("only a host can manage this play")
	}
	return nil
}
