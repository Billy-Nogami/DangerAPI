package repository

import (
	"context"
	"errors"

	"threat-service/internal/domain/threat"
)

var (
	// ErrNotFound is returned when a threat does not exist in the repository.
	ErrNotFound = errors.New("threat not found")
	// ErrAlreadyExists is returned when a threat with the same MD5 already exists.
	ErrAlreadyExists = errors.New("threat already exists")
)

// ThreatRepository persists and retrieves threat entities.
type ThreatRepository interface {
	Create(ctx context.Context, entity threat.Threat) error
	GetByMD5(ctx context.Context, md5 threat.MD5) (threat.Threat, error)
	Update(ctx context.Context, entity threat.Threat) error
}
