package model

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	PlayNameMaxRunes        = 80
	PlayDescriptionMaxRunes = 1000
)

func CleanPlayName(value *string) (*string, error) {
	return cleanOptionalPlayText(value, "name", PlayNameMaxRunes)
}

func CleanPlayDescription(value *string) (*string, error) {
	return cleanOptionalPlayText(value, "description", PlayDescriptionMaxRunes)
}

func cleanOptionalPlayText(value *string, field string, maxRunes int) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > maxRunes {
		return nil, fmt.Errorf("%s must be at most %d characters", field, maxRunes)
	}
	return &trimmed, nil
}
