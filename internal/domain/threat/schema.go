package threat

import "time"

// MD5 хранит идентификатор угрозы - хеш связанного с ней файла.
type MD5 string

// Type задаёт классификацию угрозы.
type Type string

const (
	TypeTrojan     Type = "trojan"
	TypeWorm       Type = "worm"
	TypeRansomware Type = "ransomware"
	TypeAdware     Type = "adware"
	TypeSpyware    Type = "spyware"
	TypeOther      Type = "other"
)

// Severity отражает уровень опасности угрозы.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Threat описывает угрозу, с которой работает сервис.
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

// CreateParams содержит поля, которые клиент передаёт при создании угрозы.
type CreateParams struct {
	MD5         MD5
	Name        string
	Type        Type
	Severity    Severity
	Description string
	Categories  []string
	FirstSeenAt *time.Time
}

// UpdateParams содержит изменяемые поля угрозы.
type UpdateParams struct {
	Name        string
	Type        Type
	Severity    Severity
	Description string
	Categories  []string
	FirstSeenAt *time.Time
}
