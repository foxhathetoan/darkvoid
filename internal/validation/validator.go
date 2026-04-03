package validation

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/jarviisha/darkvoid/pkg/errors"
)

var (
	// UUIDRegex validates UUID format
	UUIDRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	// SlugRegex validates URL-friendly slugs
	SlugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

	// AlphanumericRegex validates alphanumeric strings
	AlphanumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

// ValidateRequired checks if a string value is provided
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.NewValidationError(field, "required")
	}
	return nil
}

// ValidateLength validates string length
func ValidateLength(field, value string, min, max int) error {
	length := len(strings.TrimSpace(value))
	if length < min || length > max {
		return errors.NewValidationError(field, "must be between "+strconv.Itoa(min)+" and "+strconv.Itoa(max)+" characters")
	}
	return nil
}

// ValidateMinLength validates minimum string length
func ValidateMinLength(field, value string, min int) error {
	if len(strings.TrimSpace(value)) < min {
		return errors.NewValidationError(field, "must be at least "+strconv.Itoa(min)+" characters")
	}
	return nil
}

// ValidateMaxLength validates maximum string length
func ValidateMaxLength(field, value string, max int) error {
	if len(strings.TrimSpace(value)) > max {
		return errors.NewValidationError(field, "must be at most "+strconv.Itoa(max)+" characters")
	}
	return nil
}

// ValidateSlug validates URL-friendly slug format
func ValidateSlug(field, value string) error {
	if !SlugRegex.MatchString(value) {
		return errors.NewValidationError(field, "must be lowercase alphanumeric with hyphens only")
	}
	return nil
}

// ValidateAlphanumeric validates alphanumeric strings
func ValidateAlphanumeric(field, value string) error {
	if !AlphanumericRegex.MatchString(value) {
		return errors.NewValidationError(field, "must contain only letters and numbers")
	}
	return nil
}

// ValidateNoSpecialChars validates that string contains no special characters
func ValidateNoSpecialChars(field, value string) error {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsSpace(r) {
			return errors.NewValidationError(field, "must not contain special characters")
		}
	}
	return nil
}

// ValidateEnum validates that value is in allowed list
func ValidateEnum(field, value string, allowed []string) error {
	if slices.Contains(allowed, value) {
		return nil
	}
	return errors.NewValidationError(field, "invalid value")
}

// ValidatePositive validates that a number is positive
func ValidatePositive(field string, value int) error {
	if value <= 0 {
		return errors.NewValidationError(field, "must be positive")
	}
	return nil
}

// ValidateRange validates that a number is within range
func ValidateRange(field string, value, min, max int) error {
	if value < min || value > max {
		return errors.NewValidationError(field, "must be between specified range")
	}
	return nil
}
