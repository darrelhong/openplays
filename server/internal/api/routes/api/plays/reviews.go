package plays

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/reviews"
)

type GetPlayReviewsInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type PlayReviewWindow struct {
	State    string `json:"state" enum:"not_open,open,closed" doc:"Whether reviews can currently be written"`
	ClosesAt string `json:"closes_at" doc:"When the review window closes (RFC 3339)"`
}

// PlayReviewMine is the viewer's own review of one co-player. Reviews are
// only ever echoed back to their author; ratings stay anonymous to everyone
// else.
type PlayReviewMine struct {
	Rating   *int64   `json:"rating,omitempty"`
	Props    []string `json:"props"`
	Shoutout *string  `json:"shoutout,omitempty"`
}

type PlayRevieweePublic struct {
	UserID      string          `json:"user_id"`
	DisplayName string          `json:"display_name"`
	Username    *string         `json:"username,omitempty"`
	PhotoURL    *string         `json:"photo_url,omitempty"`
	IsHost      bool            `json:"is_host"`
	MyReview    *PlayReviewMine `json:"my_review,omitempty"`
}

type GetPlayReviewsOutput struct {
	Body struct {
		Window    PlayReviewWindow     `json:"window"`
		PeerProps []string             `json:"peer_props" doc:"Prop slugs available for every reviewee"`
		HostProps []string             `json:"host_props" doc:"Extra prop slugs available when the reviewee hosted"`
		Reviewees []PlayRevieweePublic `json:"reviewees"`
	}
}

type PutPlayReviewInput struct {
	ID             string `path:"id" doc:"Play ID"`
	RevieweeUserID string `path:"revieweeUserID" doc:"User ID of the co-player being reviewed"`
	Body           struct {
		Rating   *int64   `json:"rating,omitempty" minimum:"1" maximum:"5"`
		Props    []string `json:"props,omitempty"`
		Shoutout *string  `json:"shoutout,omitempty"`
	}
}

type PutPlayReviewOutput struct {
	Body PlayReviewMine
}

type PlayReviewsStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	ListReviewEligibleUsersByPlay(ctx context.Context, playID string) ([]db.ListReviewEligibleUsersByPlayRow, error)
	ListMyPlayReviews(ctx context.Context, arg db.ListMyPlayReviewsParams) ([]db.PlayReview, error)
	UpsertPlayReview(ctx context.Context, arg db.UpsertPlayReviewParams) (db.PlayReview, error)
}

func RegisterPlayReviews(api huma.API, store PlayReviewsStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "get-play-reviews",
		Summary:     "Get the viewer's review sheet for a play",
		Description: "List the co-players the authenticated user can review after the play ends, with any reviews they already wrote and the review window state. Only participants of the play can see it.",
		Method:      http.MethodGet,
		Path:        "/{id}/reviews",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *GetPlayReviewsInput) (*GetPlayReviewsOutput, error) {
		play, eligible, err := loadReviewContext(ctx, store, input.ID)
		if err != nil {
			return nil, err
		}

		user := authmw.UserFromContext(ctx)
		if !isReviewEligible(eligible, user.ID) {
			return nil, huma.Error403Forbidden("only participants can review this play")
		}

		state, closesAt := reviews.WindowState(play.EndsAt, time.Now().UTC())

		myReviews, err := store.ListMyPlayReviews(ctx, db.ListMyPlayReviewsParams{
			PlayID:         input.ID,
			ReviewerUserID: user.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get reviews")
		}
		mineByReviewee := make(map[string]*PlayReviewMine, len(myReviews))
		for _, review := range myReviews {
			mine, err := playReviewMine(review)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to read review")
			}
			mineByReviewee[review.RevieweeUserID] = mine
		}

		out := &GetPlayReviewsOutput{}
		out.Body.Window = PlayReviewWindow{State: state, ClosesAt: closesAt.Format(time.RFC3339)}
		out.Body.PeerProps = reviews.PeerPropsFor(play.Sport)
		out.Body.HostProps = reviews.HostProps
		out.Body.Reviewees = make([]PlayRevieweePublic, 0, len(eligible))
		for _, member := range eligible {
			if member.ID == user.ID {
				continue
			}
			out.Body.Reviewees = append(out.Body.Reviewees, PlayRevieweePublic{
				UserID:      member.ID,
				DisplayName: member.DisplayName,
				Username:    member.Username,
				PhotoURL:    member.PhotoUrl,
				IsHost:      member.IsHost != 0,
				MyReview:    mineByReviewee[member.ID],
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "put-play-review",
		Summary:     "Write or edit a review of a co-player",
		Description: "Save the authenticated user's review of a co-player from the same play: an anonymous star rating, prop tags, and an attributed shoutout, each optional but never all empty. Allowed from the play's end until the review window closes; reviews can be edited in that window but never deleted.",
		Method:      http.MethodPut,
		Path:        "/{id}/reviews/{revieweeUserID}",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *PutPlayReviewInput) (*PutPlayReviewOutput, error) {
		play, eligible, err := loadReviewContext(ctx, store, input.ID)
		if err != nil {
			return nil, err
		}

		user := authmw.UserFromContext(ctx)
		if !isReviewEligible(eligible, user.ID) {
			return nil, huma.Error403Forbidden("only participants can review this play")
		}
		if input.RevieweeUserID == user.ID {
			return nil, huma.Error422UnprocessableEntity("cannot review yourself")
		}
		reviewee, ok := reviewEligibleByID(eligible, input.RevieweeUserID)
		if !ok {
			return nil, huma.Error404NotFound("reviewee did not play in this game")
		}

		switch state, _ := reviews.WindowState(play.EndsAt, time.Now().UTC()); state {
		case reviews.WindowNotOpen:
			return nil, huma.Error409Conflict("reviews open once the play has ended")
		case reviews.WindowClosed:
			return nil, huma.Error409Conflict("the review window has closed")
		}

		props, err := reviews.ValidateProps(input.Body.Props, play.Sport, reviewee.IsHost != 0)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		shoutout := cleanStringPtr(input.Body.Shoutout)
		if input.Body.Rating == nil && len(props) == 0 && shoutout == nil {
			return nil, huma.Error422UnprocessableEntity("review cannot be empty")
		}

		propsJSON, err := json.Marshal(props)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to save review")
		}
		saved, err := store.UpsertPlayReview(ctx, db.UpsertPlayReviewParams{
			PlayID:         input.ID,
			ReviewerUserID: user.ID,
			RevieweeUserID: input.RevieweeUserID,
			Rating:         input.Body.Rating,
			Props:          string(propsJSON),
			Shoutout:       shoutout,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to save review")
		}

		mine, err := playReviewMine(saved)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to read review")
		}
		return &PutPlayReviewOutput{Body: *mine}, nil
	})
}

// loadReviewContext resolves the play and its review-eligible members, with
// the guards shared by both review endpoints.
func loadReviewContext(ctx context.Context, store PlayReviewsStore, playID string) (db.GetPlayByIDRow, []db.ListReviewEligibleUsersByPlayRow, error) {
	user := authmw.UserFromContext(ctx)
	if user == nil {
		return db.GetPlayByIDRow{}, nil, huma.Error401Unauthorized("not authenticated")
	}

	play, err := store.GetPlayByID(ctx, playID)
	if err == sql.ErrNoRows {
		return db.GetPlayByIDRow{}, nil, huma.Error404NotFound("play not found")
	}
	if err != nil {
		return db.GetPlayByIDRow{}, nil, huma.Error500InternalServerError("failed to get play")
	}
	if play.CancelledAt != nil {
		return db.GetPlayByIDRow{}, nil, huma.Error409Conflict("play is cancelled")
	}

	eligible, err := store.ListReviewEligibleUsersByPlay(ctx, playID)
	if err != nil {
		return db.GetPlayByIDRow{}, nil, huma.Error500InternalServerError("failed to get participants")
	}
	return play, eligible, nil
}

func isReviewEligible(eligible []db.ListReviewEligibleUsersByPlayRow, userID string) bool {
	_, ok := reviewEligibleByID(eligible, userID)
	return ok
}

func reviewEligibleByID(eligible []db.ListReviewEligibleUsersByPlayRow, userID string) (db.ListReviewEligibleUsersByPlayRow, bool) {
	for _, member := range eligible {
		if member.ID == userID {
			return member, true
		}
	}
	return db.ListReviewEligibleUsersByPlayRow{}, false
}

func playReviewMine(review db.PlayReview) (*PlayReviewMine, error) {
	props := []string{}
	if err := json.Unmarshal([]byte(review.Props), &props); err != nil {
		return nil, err
	}
	return &PlayReviewMine{
		Rating:   review.Rating,
		Props:    props,
		Shoutout: cleanStringPtr(review.Shoutout),
	}, nil
}
