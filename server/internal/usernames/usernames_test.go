package usernames

import (
	"strings"
	"testing"
)

func TestBaseFromDisplayNameUsesUnderscores(t *testing.T) {
	got := BaseFromDisplayName(" Darrel Hong Jr. ")
	if got != "darrel_hong_jr" {
		t.Fatalf("BaseFromDisplayName = %q, want darrel_hong_jr", got)
	}
}

func TestBaseFromDisplayNameFallsBackToPlayer(t *testing.T) {
	got := BaseFromDisplayName("名字")
	if got != "player" {
		t.Fatalf("BaseFromDisplayName = %q, want player", got)
	}
}

func TestNormalize(t *testing.T) {
	got, err := Normalize(" Seed_Player ")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got != "seed_player" {
		t.Fatalf("Normalize = %q, want seed_player", got)
	}

	if _, err := Normalize("seed-player"); err == nil {
		t.Fatal("Normalize accepted hyphenated username")
	}
}

func TestWithRandomSuffix(t *testing.T) {
	got, err := WithRandomSuffix(strings.Repeat("a", MaxLength))
	if err != nil {
		t.Fatalf("WithRandomSuffix: %v", err)
	}
	if len(got) != MaxLength {
		t.Fatalf("len = %d, want %d", len(got), MaxLength)
	}
	if got[MaxLength-SuffixLength-1] != '_' {
		t.Fatalf("suffix separator missing in %q", got)
	}
}
