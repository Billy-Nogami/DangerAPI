package repository

import (
	"context"
	"errors"

	"threat-service/internal/domain/threat"
)

var (
	// ErrNotFound возвращается, когда угроза отсутствует в хранилище.
	ErrNotFound = errors.New("threat not found")
	// ErrAlreadyExists возвращается при попытке создать угрозу с существующим MD5.
	ErrAlreadyExists = errors.New("threat already exists")
)

// ThreatRepository сохраняет угрозы и ищет их в хранилище.
type ThreatRepository interface {
	Create(ctx context.Context, entity threat.Threat) error
	GetByMD5(ctx context.Context, md5 threat.MD5) (threat.Threat, error)
	Update(ctx context.Context, entity threat.Threat) error
}
