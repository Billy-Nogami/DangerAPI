package threat

import "time"

// MD5 identifies a threat by the MD5 hash of the related file.
type MD5 string

// Type classifies the threat.
type Type string

const (
	TypeTrojan     Type = "trojan"
	TypeWorm       Type = "worm"
	TypeRansomware Type = "ransomware"
	TypeAdware     Type = "adware"
	TypeSpyware    Type = "spyware"
	TypeOther      Type = "other"
)

// Severity describes the impact of a threat.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Threat is the domain entity stored by the service.
type Threat struct {
	MD5         MD5
	Name        string
	Type        Type
	Severity    Severity
	Description string
	Categories  []string
	FirstSeenAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateParams contains client-controlled fields for a new threat.
type CreateParams struct {
	MD5         MD5
	Name        string
	Type        Type
	Severity    Severity
	Description string
	Categories  []string
	FirstSeenAt *time.Time
}

// UpdateParams contains the mutable fields of a threat.
type UpdateParams struct {
	Name        string
	Type        Type
	Severity    Severity
	Description string
	Categories  []string
	FirstSeenAt *time.Time
}
