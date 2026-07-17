package model

import "testing"

func TestProfileLinksStringNormalizesIdentifiers(t *testing.T) {
	telegram := "  @openplays_sg "
	stravaID := " 123456 "
	raw, err := ProfileLinksString(&ProfileLinks{
		Telegram:        &telegram,
		StravaAthleteID: &stravaID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if raw == nil {
		t.Fatal("raw = nil")
	}

	parsed, err := ParseProfileLinks(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed == nil || parsed.Telegram == nil || *parsed.Telegram != "openplays_sg" {
		t.Fatalf("telegram = %#v", parsed)
	}
	if parsed.StravaAthleteID == nil || *parsed.StravaAthleteID != "123456" {
		t.Fatalf("strava athlete ID = %#v", parsed.StravaAthleteID)
	}
}

func TestProfileLinksStringRejectsInvalidIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		links ProfileLinks
	}{
		{"path separator", ProfileLinks{Rovo: stringPtr("name/other")}},
		{"telegram dot", ProfileLinks{Telegram: stringPtr("name.other")}},
		{"x too long", ProfileLinks{X: stringPtr("1234567890123456")}},
		{"strava username", ProfileLinks{StravaAthleteID: stringPtr("runner")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ProfileLinksString(&tt.links); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestProfileLinksStringClearsEmptyValues(t *testing.T) {
	empty := " @ "
	raw, err := ProfileLinksString(&ProfileLinks{Instagram: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if raw != nil {
		t.Fatalf("raw = %q, want nil", *raw)
	}
}

func TestNormalizeBio(t *testing.T) {
	bio := "  Always up for a game.  "
	got, err := NormalizeBio(&bio)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != "Always up for a game." {
		t.Fatalf("bio = %#v", got)
	}

	tooLong := make([]rune, MaxBioLength+1)
	for i := range tooLong {
		tooLong[i] = '界'
	}
	value := string(tooLong)
	if _, err := NormalizeBio(&value); err == nil {
		t.Fatal("expected length validation error")
	}
}

func stringPtr(value string) *string { return &value }
