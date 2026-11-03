package usecase

import (
	"context"
	"time"

	"threat-service/internal/domain/threat"
	"threat-service/internal/repository"
)

// ThreatService implements application scenarios for threat records.
type ThreatService struct {
	repository repository.ThreatRepository
	clock      Clock
}

// NewThreatService creates a service with its required dependencies.
func NewThreatService(repository repository.ThreatRepository, clock Clock) *ThreatService {
	return &ThreatService{
		repository: repository,
		clock:      clock,
	}
}

// Create validates and stores a new threat.
func (s *ThreatService) Create(ctx context.Context, params threat.CreateParams) (threat.Threat, error) {
	if err := params.Validate(); err != nil {
		return threat.Threat{}, err
	}

	now := s.clock.Now().UTC()
	entity := threat.Threat{
		MD5:         threat.NormalizeMD5(params.MD5),
		Name:        params.Name,
		Type:        params.Type,
		Severity:    params.Severity,
		Description: params.Description,
		Categories:  copyStrings(params.Categories),
		FirstSeenAt: copyTime(params.FirstSeenAt),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := entity.Validate(); err != nil {
		return threat.Threat{}, err
	}
	if err := s.repository.Create(ctx, entity); err != nil {
		return threat.Threat{}, err
	}

	return entity, nil
}

// GetByMD5 returns a threat identified by a valid MD5 hash.
func (s *ThreatService) GetByMD5(ctx context.Context, md5 threat.MD5) (threat.Threat, error) {
	if err := threat.ValidateMD5(md5); err != nil {
		return threat.Threat{}, err
	}

	return s.repository.GetByMD5(ctx, threat.NormalizeMD5(md5))
}

// Update replaces all mutable fields of an existing threat.
func (s *ThreatService) Update(ctx context.Context, md5 threat.MD5, params threat.UpdateParams) (threat.Threat, error) {
	if err := threat.ValidateMD5(md5); err != nil {
		return threat.Threat{}, err
	}
	if err := params.Validate(); err != nil {
		return threat.Threat{}, err
	}

	entity, err := s.repository.GetByMD5(ctx, threat.NormalizeMD5(md5))
	if err != nil {
		return threat.Threat{}, err
	}

	entity.Name = params.Name
	entity.Type = params.Type
	entity.Severity = params.Severity
	entity.Description = params.Description
	entity.Categories = copyStrings(params.Categories)
	entity.FirstSeenAt = copyTime(params.FirstSeenAt)
	entity.UpdatedAt = s.clock.Now().UTC()

	if err := entity.Validate(); err != nil {
		return threat.Threat{}, err
	}
	if err := s.repository.Update(ctx, entity); err != nil {
		return threat.Threat{}, err
	}

	return entity, nil
}

func copyStrings(values []string) []string {
	return append([]string(nil), values...)
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	copy := *value
	return &copy
}
