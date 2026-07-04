package plays

import (
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
)

// requireRosterOpen rejects roster mutations once a play has ended. The final
// roster is the record of who played and feeds post-game reviews, so it is
// frozen at ends_at.
func requireRosterOpen(play db.GetPlayByIDRow) error {
	if !play.EndsAt.After(time.Now().UTC()) {
		return huma.Error409Conflict("play has ended")
	}
	return nil
}
