package reviews

import (
	"slices"
	"testing"
	"time"

	"openplays/server/internal/model"
)

func TestWindowState(t *testing.T) {
	endsAt := time.Date(2026, 7, 1, 6, 0, 0, 0, time.UTC)
	wantClosesAt := endsAt.Add(Window)

	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"before the play ends", endsAt.Add(-time.Minute), WindowNotOpen},
		{"exactly at ends_at", endsAt, WindowOpen},
		{"during the window", endsAt.Add(7 * 24 * time.Hour), WindowOpen},
		{"exactly at close", wantClosesAt, WindowOpen},
		{"after the window", wantClosesAt.Add(time.Second), WindowClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, closesAt := WindowState(endsAt, tc.now)
			if state != tc.want {
				t.Fatalf("state = %q, want %q", state, tc.want)
			}
			if !closesAt.Equal(wantClosesAt) {
				t.Fatalf("closesAt = %v, want %v", closesAt, wantClosesAt)
			}
		})
	}
}

func TestWindowStateDevBypass(t *testing.T) {
	t.Setenv("DEV_REVIEWS_ALWAYS_OPEN", "true")

	endsAt := time.Date(2026, 7, 1, 6, 0, 0, 0, time.UTC)
	if state, _ := WindowState(endsAt, endsAt.Add(-time.Hour)); state != WindowOpen {
		t.Fatalf("state before end = %q, want open under dev bypass", state)
	}
	if state, _ := WindowState(endsAt, endsAt.Add(Window+time.Hour)); state != WindowOpen {
		t.Fatalf("state after close = %q, want open under dev bypass", state)
	}
}

func TestValidateProps(t *testing.T) {
	t.Run("accepts universal and same-sport props", func(t *testing.T) {
		got, err := ValidateProps([]string{"great_sport", "powerful_smash"}, model.SportBadminton, false)
		if err != nil {
			t.Fatalf("ValidateProps: %v", err)
		}
		if !slices.Equal(got, []string{"great_sport", "powerful_smash"}) {
			t.Fatalf("props = %v", got)
		}
	})

	t.Run("rejects another sport's props", func(t *testing.T) {
		if _, err := ValidateProps([]string{"big_serve"}, model.SportBadminton, false); err == nil {
			t.Fatal("want error for tennis prop on a badminton play")
		}
	})

	t.Run("dedupes preserving order", func(t *testing.T) {
		got, err := ValidateProps([]string{"chill_vibes", "great_sport", "chill_vibes"}, model.SportTennis, false)
		if err != nil {
			t.Fatalf("ValidateProps: %v", err)
		}
		if !slices.Equal(got, []string{"chill_vibes", "great_sport"}) {
			t.Fatalf("props = %v", got)
		}
	})

	t.Run("host props require a host reviewee", func(t *testing.T) {
		if _, err := ValidateProps([]string{"well_organized"}, model.SportBadminton, false); err == nil {
			t.Fatal("want error for host prop on non-host")
		}
		got, err := ValidateProps([]string{"well_organized", "great_sport"}, model.SportBadminton, true)
		if err != nil {
			t.Fatalf("ValidateProps: %v", err)
		}
		if !slices.Equal(got, []string{"well_organized", "great_sport"}) {
			t.Fatalf("props = %v", got)
		}
	})

	t.Run("rejects unknown slugs", func(t *testing.T) {
		if _, err := ValidateProps([]string{"free_beer"}, model.SportBadminton, true); err == nil {
			t.Fatal("want error for unknown prop")
		}
	})

	t.Run("caps props per review", func(t *testing.T) {
		three := []string{"great_sport", "humble", "punctual"}
		if _, err := ValidateProps(three, model.SportBadminton, false); err == nil {
			t.Fatal("want error for more than MaxPropsPerReview props")
		}
		// Duplicates collapse before the cap applies
		got, err := ValidateProps([]string{"great_sport", "humble", "great_sport"}, model.SportBadminton, false)
		if err != nil {
			t.Fatalf("ValidateProps: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("props = %v, want 2", got)
		}
	})

	t.Run("empty input is fine", func(t *testing.T) {
		got, err := ValidateProps(nil, model.SportBadminton, false)
		if err != nil {
			t.Fatalf("ValidateProps: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("props = %v, want empty", got)
		}
	})
}

func TestPeerPropsFor(t *testing.T) {
	badminton := PeerPropsFor(model.SportBadminton)
	if !slices.Contains(badminton, "great_sport") || !slices.Contains(badminton, "powerful_smash") {
		t.Fatalf("badminton props = %v", badminton)
	}
	if slices.Contains(badminton, "big_serve") {
		t.Fatalf("badminton props leaked tennis slugs: %v", badminton)
	}
	// Unknown sports still offer the universal set
	unknown := PeerPropsFor(model.Sport("chess"))
	if !slices.Equal(unknown, UniversalPeerProps) {
		t.Fatalf("unknown sport props = %v, want universal only", unknown)
	}
}
