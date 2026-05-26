package plays_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/plays"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func setupGetDetailTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterGet(grp, store, authmw.OptionalAuth(api, svc))

	return httptest.NewServer(r)
}

func TestGetPlayDetail_VisibilityAndDerivedCounts(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-creator-1")
	confirmedID := createRouteTestUser(t, ctx, queries, "detail-confirmed-1")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))

	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, confirmedID)
	guestName := "Guest Wait"
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    playID,
		GuestName: &guestName,
		Status:    model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant waitlist guest: %v", err)
	}

	ts := setupGetDetailTest(&fakeAuthStore{sessionErr: context.Canceled}, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		ConfirmedParticipants []struct {
			DisplayName *string `json:"display_name"`
		} `json:"confirmed_participants"`
		Waitlist []struct {
			DisplayName *string `json:"display_name"`
		} `json:"waitlist"`
		ViewerState    *string `json:"viewer_state"`
		CanManage      *bool   `json:"can_manage"`
		ConfirmedCount *int64  `json:"confirmed_count"`
		WaitlistCount  *int64  `json:"waitlist_count"`
		SlotsLeft      *int64  `json:"slots_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(out.ConfirmedParticipants) != 2 {
		t.Fatalf("confirmed_participants len = %d, want 2", len(out.ConfirmedParticipants))
	}
	if len(out.Waitlist) != 1 {
		t.Fatalf("waitlist len = %d, want 1", len(out.Waitlist))
	}
	if out.Waitlist[0].DisplayName == nil || *out.Waitlist[0].DisplayName != guestName {
		t.Fatalf("waitlist[0].display_name = %v, want %q", out.Waitlist[0].DisplayName, guestName)
	}
	if out.ConfirmedCount == nil || *out.ConfirmedCount != 2 {
		t.Fatalf("confirmed_count = %v, want 2", out.ConfirmedCount)
	}
	if out.WaitlistCount == nil || *out.WaitlistCount != 1 {
		t.Fatalf("waitlist_count = %v, want 1", out.WaitlistCount)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 2 {
		t.Fatalf("slots_left = %v, want 2", out.SlotsLeft)
	}
	if out.ViewerState == nil || *out.ViewerState != "not_joined" {
		t.Fatalf("viewer_state = %v, want not_joined", out.ViewerState)
	}
	if out.CanManage == nil || *out.CanManage {
		t.Fatalf("can_manage = %v, want false", out.CanManage)
	}
}

func TestGetPlayDetail_CreatorPermissions(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-creator-2")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)

	authStore := sessionWithProfile(creatorID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupGetDetailTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		ViewerState *string `json:"viewer_state"`
		CanManage   *bool   `json:"can_manage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ViewerState == nil || *out.ViewerState != "creator" {
		t.Fatalf("viewer_state = %v, want creator", out.ViewerState)
	}
	if out.CanManage == nil || !*out.CanManage {
		t.Fatalf("can_manage = %v, want true", out.CanManage)
	}
}

func TestGetPlayDetail_ViewerStateByParticipantStatus(t *testing.T) {
	tests := []struct {
		name          string
		participantSt model.PlayParticipantStatus
		wantState     string
	}{
		{name: "confirmed", participantSt: model.ParticipantConfirmed, wantState: "confirmed"},
		{name: "waitlisted", participantSt: model.ParticipantWaitlisted, wantState: "waitlisted"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := testdb.New(t)
			queries := db.New(sqlDB)
			ctx := context.Background()

			creatorID := createRouteTestUser(t, ctx, queries, "detail-creator-"+tc.name)
			viewerID := createRouteTestUser(t, ctx, queries, "detail-viewer-"+tc.name)
			playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
			seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
			if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
				PlayID: playID,
				UserID: &viewerID,
				Status: tc.participantSt,
			}); err != nil {
				t.Fatalf("CreatePlayParticipant viewer: %v", err)
			}

			authStore := sessionWithProfile(viewerID, ptrString(`{"badminton":{"level":"HB"}}`))
			ts := setupGetDetailTest(authStore, queries)
			defer ts.Close()

			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
			req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}

			var out struct {
				ViewerState *string `json:"viewer_state"`
				CanManage   *bool   `json:"can_manage"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.ViewerState == nil || *out.ViewerState != tc.wantState {
				t.Fatalf("viewer_state = %v, want %s", out.ViewerState, tc.wantState)
			}
			if out.CanManage == nil || *out.CanManage {
				t.Fatalf("can_manage = %v, want false", out.CanManage)
			}
		})
	}
}

func init() {
	_ = time.Now
}
