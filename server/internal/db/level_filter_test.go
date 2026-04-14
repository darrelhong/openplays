package db_test

import (
	"context"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

// Helper to create a play with specific level ordinals.
func insertPlay(t *testing.T, queries *db.Queries, host string, levelMinOrd, levelMaxOrd *int64) {
	t.Helper()
	source := "telegram"
	startsAt := time.Now().UTC().Add(24 * time.Hour)
	venueID := int64(1)
	_, err := queries.UpsertPlay(context.Background(), db.UpsertPlayParams{
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    host,
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "Test Venue",
		VenueID:     &venueID,
		LevelMinOrd: levelMinOrd,
		LevelMaxOrd: levelMaxOrd,
		Currency:    "SGD",
		Source:      &source,
	})
	if err != nil {
		t.Fatalf("insertPlay(%s): %v", host, err)
	}
}

func intPtr(v int64) *int64 { return &v }

// listWithLevel queries ListUpcomingPlays with the given level filter.
func listWithLevel(t *testing.T, queries *db.Queries, filterMin, filterMax interface{}) []db.ListUpcomingPlaysRow {
	t.Helper()
	rows, err := queries.ListUpcomingPlays(context.Background(), db.ListUpcomingPlaysParams{
		FilterLevelMinOrd: filterMin,
		FilterLevelMaxOrd: filterMax,
		PageSize:          100,
	})
	if err != nil {
		t.Fatalf("ListUpcomingPlays: %v", err)
	}
	return rows
}

func TestLevelFilter_MinOnly(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	// Ordinals: LB=10, MB=20, HB=30, LI=40, MI=50, HI=60, A=70
	insertPlay(t, queries, "host-lb-hb", intPtr(10), intPtr(30)) // LB-HB
	insertPlay(t, queries, "host-hb-li", intPtr(30), intPtr(40)) // HB-LI
	insertPlay(t, queries, "host-li-hi", intPtr(40), intPtr(60)) // LI-HI
	insertPlay(t, queries, "host-no-level", nil, nil)            // sell_booking, no level

	// Filter: level_min=HB (ord 30), no max — "HB and above"
	// Should match: HB-LI (max 40 >= 30), LI-HI (max 60 >= 30), no-level (null passes)
	// Should NOT match: LB-HB? Actually LB-HB max=30 >= 30, so it matches too.
	rows := listWithLevel(t, queries, 30, nil)

	hosts := make(map[string]bool)
	for _, r := range rows {
		hosts[r.HostName] = true
	}

	if len(rows) != 4 {
		t.Errorf("level_min=HB: got %d plays, want 4", len(rows))
	}
	if !hosts["host-lb-hb"] {
		t.Error("expected host-lb-hb (LB-HB includes HB)")
	}
	if !hosts["host-hb-li"] {
		t.Error("expected host-hb-li")
	}
	if !hosts["host-li-hi"] {
		t.Error("expected host-li-hi")
	}
	if !hosts["host-no-level"] {
		t.Error("expected host-no-level (null levels always pass)")
	}
}

func TestLevelFilter_MaxOnly(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	insertPlay(t, queries, "host-lb-hb", intPtr(10), intPtr(30)) // LB-HB
	insertPlay(t, queries, "host-hb-li", intPtr(30), intPtr(40)) // HB-LI
	insertPlay(t, queries, "host-li-hi", intPtr(40), intPtr(60)) // LI-HI
	insertPlay(t, queries, "host-no-level", nil, nil)

	// Filter: level_max=HB (ord 30), no min — "HB and below"
	// Should match: LB-HB (min 10 <= 30), HB-LI (min 30 <= 30), no-level
	// Should NOT match: LI-HI (min 40 > 30)
	rows := listWithLevel(t, queries, nil, 30)

	hosts := make(map[string]bool)
	for _, r := range rows {
		hosts[r.HostName] = true
	}

	if len(rows) != 3 {
		t.Errorf("level_max=HB: got %d plays, want 3", len(rows))
	}
	if !hosts["host-lb-hb"] {
		t.Error("expected host-lb-hb")
	}
	if !hosts["host-hb-li"] {
		t.Error("expected host-hb-li")
	}
	if hosts["host-li-hi"] {
		t.Error("did not expect host-li-hi (LI-HI min 40 > 30)")
	}
	if !hosts["host-no-level"] {
		t.Error("expected host-no-level")
	}
}

func TestLevelFilter_Range(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	insertPlay(t, queries, "host-lb-mb", intPtr(10), intPtr(20)) // LB-MB
	insertPlay(t, queries, "host-mb-hb", intPtr(20), intPtr(30)) // MB-HB
	insertPlay(t, queries, "host-hb-li", intPtr(30), intPtr(40)) // HB-LI
	insertPlay(t, queries, "host-li-hi", intPtr(40), intPtr(60)) // LI-HI
	insertPlay(t, queries, "host-hi-a", intPtr(60), intPtr(70))  // HI-A
	insertPlay(t, queries, "host-no-level", nil, nil)

	// Filter: level_min=HB (30), level_max=MI (50) — "plays overlapping HB-MI"
	// Overlap condition: play.max >= filter.min AND play.min <= filter.max
	// MB-HB: max 30 >= 30 AND min 20 <= 50 → yes
	// HB-LI: max 40 >= 30 AND min 30 <= 50 → yes
	// LI-HI: max 60 >= 30 AND min 40 <= 50 → yes
	// LB-MB: max 20 >= 30? No → excluded
	// HI-A: max 70 >= 30 AND min 60 <= 50? No → excluded
	// no-level: null passes both → yes
	rows := listWithLevel(t, queries, 30, 50)

	hosts := make(map[string]bool)
	for _, r := range rows {
		hosts[r.HostName] = true
	}

	if len(rows) != 4 {
		t.Errorf("level HB-MI: got %d plays, want 4", len(rows))
		for _, r := range rows {
			t.Logf("  got: %s", r.HostName)
		}
	}
	if hosts["host-lb-mb"] {
		t.Error("did not expect host-lb-mb (LB-MB max 20 < filter min 30)")
	}
	if !hosts["host-mb-hb"] {
		t.Error("expected host-mb-hb")
	}
	if !hosts["host-hb-li"] {
		t.Error("expected host-hb-li")
	}
	if !hosts["host-li-hi"] {
		t.Error("expected host-li-hi")
	}
	if hosts["host-hi-a"] {
		t.Error("did not expect host-hi-a (HI-A min 60 > filter max 50)")
	}
	if !hosts["host-no-level"] {
		t.Error("expected host-no-level")
	}
}

func TestLevelFilter_NoFilter(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	insertPlay(t, queries, "host-lb-hb", intPtr(10), intPtr(30))
	insertPlay(t, queries, "host-li-hi", intPtr(40), intPtr(60))
	insertPlay(t, queries, "host-no-level", nil, nil)

	// No level filter — should return all
	rows := listWithLevel(t, queries, nil, nil)

	if len(rows) != 3 {
		t.Errorf("no filter: got %d plays, want 3", len(rows))
	}
}

func TestLevelFilter_ExactLevel(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	insertPlay(t, queries, "host-hb-hb", intPtr(30), intPtr(30)) // exactly HB
	insertPlay(t, queries, "host-hb-li", intPtr(30), intPtr(40)) // HB-LI
	insertPlay(t, queries, "host-li-li", intPtr(40), intPtr(40)) // exactly LI
	insertPlay(t, queries, "host-no-level", nil, nil)

	// Filter: level_min=HB, level_max=HB — "exactly HB"
	// HB-HB: max 30 >= 30 AND min 30 <= 30 → yes
	// HB-LI: max 40 >= 30 AND min 30 <= 30 → yes (range includes HB)
	// LI-LI: max 40 >= 30 AND min 40 <= 30? No → excluded
	// no-level: yes
	rows := listWithLevel(t, queries, 30, 30)

	hosts := make(map[string]bool)
	for _, r := range rows {
		hosts[r.HostName] = true
	}

	if len(rows) != 3 {
		t.Errorf("exact HB: got %d plays, want 3", len(rows))
	}
	if !hosts["host-hb-hb"] {
		t.Error("expected host-hb-hb")
	}
	if !hosts["host-hb-li"] {
		t.Error("expected host-hb-li (range includes HB)")
	}
	if hosts["host-li-li"] {
		t.Error("did not expect host-li-li")
	}
}

func TestLevelFilter_NullMaxOnPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	// Play with "LI & above" — min=40, max=NULL
	insertPlay(t, queries, "host-li-above", intPtr(40), nil)
	insertPlay(t, queries, "host-hb-li", intPtr(30), intPtr(40))

	// Filter: level_min=HI (60)
	// host-li-above: max NULL passes, included
	// host-hb-li: max 40 >= 60? No → excluded
	rows := listWithLevel(t, queries, 60, nil)

	hosts := make(map[string]bool)
	for _, r := range rows {
		hosts[r.HostName] = true
	}

	if !hosts["host-li-above"] {
		t.Error("expected host-li-above (null max = open-ended)")
	}
	if hosts["host-hb-li"] {
		t.Error("did not expect host-hb-li")
	}
}
