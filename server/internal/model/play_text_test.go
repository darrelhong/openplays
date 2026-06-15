package model

import (
	"strings"
	"testing"
)

func TestCleanPlayDescription(t *testing.T) {
	t.Run("trims non-empty text", func(t *testing.T) {
		value := "  Bring water and shuttles.  "
		got, err := CleanPlayDescription(&value)
		if err != nil {
			t.Fatalf("CleanPlayDescription: %v", err)
		}
		if got == nil || *got != "Bring water and shuttles." {
			t.Fatalf("CleanPlayDescription = %v, want trimmed description", got)
		}
	})

	t.Run("blank text clears the field", func(t *testing.T) {
		value := "   "
		got, err := CleanPlayDescription(&value)
		if err != nil {
			t.Fatalf("CleanPlayDescription: %v", err)
		}
		if got != nil {
			t.Fatalf("CleanPlayDescription = %v, want nil", got)
		}
	})

	t.Run("accepts one thousand runes", func(t *testing.T) {
		value := strings.Repeat("界", PlayDescriptionMaxRunes)
		got, err := CleanPlayDescription(&value)
		if err != nil {
			t.Fatalf("CleanPlayDescription: %v", err)
		}
		if got == nil || *got != value {
			t.Fatalf("CleanPlayDescription did not preserve max length value")
		}
	})

	t.Run("rejects more than one thousand runes", func(t *testing.T) {
		value := strings.Repeat("a", PlayDescriptionMaxRunes+1)
		got, err := CleanPlayDescription(&value)
		if err == nil {
			t.Fatalf("CleanPlayDescription = %v, want error", got)
		}
		if want := "description must be at most 1000 characters"; err.Error() != want {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	})
}

func TestCleanPlayNameRejectsMoreThanEightyRunes(t *testing.T) {
	value := strings.Repeat("a", PlayNameMaxRunes+1)
	got, err := CleanPlayName(&value)
	if err == nil {
		t.Fatalf("CleanPlayName = %v, want error", got)
	}
}
