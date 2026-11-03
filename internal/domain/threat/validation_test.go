package threat

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateMD5(t *testing.T) {
	t.Parallel()

	validMD5 := MD5("d41d8cd98f00b204e9800998ecf8427e")
	testCases := []struct {
		name    string
		value   MD5
		wantErr bool
	}{
		{name: "lowercase hexadecimal", value: validMD5},
		{name: "uppercase hexadecimal", value: MD5(strings.ToUpper(string(validMD5)))},
		{name: "too short", value: "d41d8cd98f00b204e9800998ecf8427", wantErr: true},
		{name: "non hexadecimal character", value: "d41d8cd98f00b204e9800998ecf8427g", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateMD5(testCase.value)
			if testCase.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestNormalizeMD5(t *testing.T) {
	t.Parallel()

	actual := NormalizeMD5("D41D8CD98F00B204E9800998ECF8427E")
	const expected = MD5("d41d8cd98f00b204e9800998ecf8427e")
	if actual != expected {
		t.Fatalf("NormalizeMD5() = %q, want %q", actual, expected)
	}
}

func TestCreateParamsValidate(t *testing.T) {
	t.Parallel()

	valid := validCreateParams()
	testCases := []struct {
		name      string
		mutate    func(*CreateParams)
		wantField string
	}{
		{name: "empty name", mutate: func(p *CreateParams) { p.Name = "  " }, wantField: "name"},
		{name: "unknown type", mutate: func(p *CreateParams) { p.Type = "virus" }, wantField: "type"},
		{name: "unknown severity", mutate: func(p *CreateParams) { p.Severity = "urgent" }, wantField: "severity"},
		{name: "long description", mutate: func(p *CreateParams) { p.Description = strings.Repeat("a", maxDescriptionLength+1) }, wantField: "description"},
		{name: "duplicate category", mutate: func(p *CreateParams) { p.Categories = []string{"banker", "banker"} }, wantField: "categories"},
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("valid parameters failed validation: %v", err)
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			params := valid
			testCase.mutate(&params)

			assertValidationField(t, params.Validate(), testCase.wantField)
		})
	}
}

func TestUpdateParamsValidate(t *testing.T) {
	t.Parallel()

	params := UpdateParams{
		Name:       "Trojan.Win32.Generic",
		Type:       TypeTrojan,
		Severity:   SeverityHigh,
		Categories: []string{"banker"},
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("valid parameters failed validation: %v", err)
	}

	params.Categories = []string{""}
	assertValidationField(t, params.Validate(), "categories")
}

func TestThreatValidate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 3, 10, 0, 0, 0, time.UTC)
	firstSeen := now.Add(-time.Hour)
	threat := Threat{
		MD5:         "d41d8cd98f00b204e9800998ecf8427e",
		Name:        "Trojan.Win32.Generic",
		Type:        TypeTrojan,
		Severity:    SeverityHigh,
		Categories:  []string{"banker"},
		FirstSeenAt: &firstSeen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := threat.Validate(); err != nil {
		t.Fatalf("valid threat failed validation: %v", err)
	}

	threat.UpdatedAt = now.Add(-time.Second)
	assertValidationField(t, threat.Validate(), "updatedAt")
}

func validCreateParams() CreateParams {
	return CreateParams{
		MD5:         "d41d8cd98f00b204e9800998ecf8427e",
		Name:        "Trojan.Win32.Generic",
		Type:        TypeTrojan,
		Severity:    SeverityHigh,
		Description: "Credential-stealing Trojan.",
		Categories:  []string{"banker", "stealer"},
	}
}

func assertValidationField(t *testing.T, err error, wantField string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected validation error")
	}

	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %T (%v)", err, err)
	}
	if validationError.Field != wantField {
		t.Fatalf("validation error field = %q, want %q", validationError.Field, wantField)
	}
}
