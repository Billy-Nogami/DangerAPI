package threat

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxNameLength        = 255
	maxDescriptionLength = 4000
	maxCategories        = 20
	maxCategoryLength    = 100
)

// ValidationError identifies a field that does not meet domain requirements.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

// NormalizeMD5 converts a validated MD5 value to its canonical representation.
func NormalizeMD5(value MD5) MD5 {
	return MD5(strings.ToLower(string(value)))
}

// Validate checks whether a new threat can be created.
func (p CreateParams) Validate() error {
	if err := ValidateMD5(p.MD5); err != nil {
		return err
	}

	return validateMutableFields(
		p.Name,
		p.Type,
		p.Severity,
		p.Description,
		p.Categories,
	)
}

// Validate checks whether the mutable part of a threat can be replaced.
func (p UpdateParams) Validate() error {
	return validateMutableFields(
		p.Name,
		p.Type,
		p.Severity,
		p.Description,
		p.Categories,
	)
}

// Validate checks the complete domain entity, including server-controlled dates.
func (t Threat) Validate() error {
	if err := (CreateParams{
		MD5:         t.MD5,
		Name:        t.Name,
		Type:        t.Type,
		Severity:    t.Severity,
		Description: t.Description,
		Categories:  t.Categories,
		FirstSeenAt: t.FirstSeenAt,
	}).Validate(); err != nil {
		return err
	}

	if t.CreatedAt.IsZero() {
		return validationError("createdAt", "must be set")
	}
	if t.UpdatedAt.IsZero() {
		return validationError("updatedAt", "must be set")
	}
	if t.UpdatedAt.Before(t.CreatedAt) {
		return validationError("updatedAt", "must not be earlier than createdAt")
	}
	if t.FirstSeenAt != nil && t.FirstSeenAt.After(t.UpdatedAt) {
		return validationError("firstSeenAt", "must not be later than updatedAt")
	}

	return nil
}

// ValidateMD5 checks that a value consists of exactly 32 hexadecimal symbols.
func ValidateMD5(value MD5) error {
	if len(value) != 32 {
		return validationError("md5", "must contain exactly 32 hexadecimal characters")
	}

	for _, char := range value {
		if !isHexadecimal(char) {
			return validationError("md5", "must contain only hexadecimal characters")
		}
	}

	return nil
}

func validateMutableFields(name string, threatType Type, severity Severity, description string, categories []string) error {
	if trimmedName := strings.TrimSpace(name); trimmedName == "" {
		return validationError("name", "must not be empty")
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return validationError("name", "must not exceed 255 characters")
	}
	if !isKnownType(threatType) {
		return validationError("type", "must be a supported threat type")
	}
	if !isKnownSeverity(severity) {
		return validationError("severity", "must be a supported severity")
	}
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return validationError("description", "must not exceed 4000 characters")
	}
	if len(categories) > maxCategories {
		return validationError("categories", "must not contain more than 20 items")
	}

	seen := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		if strings.TrimSpace(category) == "" {
			return validationError("categories", "must not contain empty values")
		}
		if utf8.RuneCountInString(category) > maxCategoryLength {
			return validationError("categories", "items must not exceed 100 characters")
		}
		if _, exists := seen[category]; exists {
			return validationError("categories", "must not contain duplicate values")
		}
		seen[category] = struct{}{}
	}

	return nil
}

func isHexadecimal(char rune) bool {
	return char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F'
}

func isKnownType(value Type) bool {
	switch value {
	case TypeTrojan, TypeWorm, TypeRansomware, TypeAdware, TypeSpyware, TypeOther:
		return true
	default:
		return false
	}
}

func isKnownSeverity(value Severity) bool {
	switch value {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	default:
		return false
	}
}

func validationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
