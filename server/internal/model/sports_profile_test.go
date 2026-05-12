package model

import "testing"

func TestParseSportsProfileValidatesAndNormalizesLevels(t *testing.T) {
	raw := `{"badminton":{"level":" LI "},"tennis":{"level":"3.5"}}`

	profile, err := ParseSportsProfile(&raw)
	if err != nil {
		t.Fatalf("ParseSportsProfile: %v", err)
	}
	if got := profile.LevelFor(SportBadminton); got == nil || *got != "LI" {
		t.Fatalf("badminton level = %v, want LI", got)
	}
	if got := profile.LevelFor(SportTennis); got == nil || *got != "3.5" {
		t.Fatalf("tennis level = %v, want 3.5", got)
	}
}

func TestParseSportsProfileRejectsUnsupportedLevel(t *testing.T) {
	raw := `{"badminton":{"level":"ZZ"}}`

	_, err := ParseSportsProfile(&raw)
	if err == nil {
		t.Fatal("expected invalid badminton level error")
	}
}

func TestSportsProfileStringRoundTripAndEmptyProfile(t *testing.T) {
	badminton := "HB"
	profile := &SportsProfile{
		Badminton: &SportLevelProfile{Level: &badminton},
	}

	raw, err := SportsProfileString(profile)
	if err != nil {
		t.Fatalf("SportsProfileString: %v", err)
	}
	if raw == nil || *raw == "" {
		t.Fatal("expected non-empty raw profile JSON")
	}

	roundTrip, err := ParseSportsProfile(raw)
	if err != nil {
		t.Fatalf("ParseSportsProfile round trip: %v", err)
	}
	if got := roundTrip.LevelFor(SportBadminton); got == nil || *got != "HB" {
		t.Fatalf("round-trip badminton level = %v, want HB", got)
	}

	emptyLevel := " "
	emptyRaw, err := SportsProfileString(&SportsProfile{
		Tennis: &SportLevelProfile{Level: &emptyLevel},
	})
	if err != nil {
		t.Fatalf("SportsProfileString empty: %v", err)
	}
	if emptyRaw != nil {
		t.Fatalf("empty profile raw = %q, want nil", *emptyRaw)
	}
}
