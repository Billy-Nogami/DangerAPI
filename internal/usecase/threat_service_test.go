package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"threat-service/internal/domain/threat"
	"threat-service/internal/repository"
)

func TestThreatServiceCreate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 3, 10, 0, 0, 0, time.UTC)
	repo := newMemoryThreatRepository()
	service := NewThreatService(repo, fixedClock{now: now})
	firstSeen := now.Add(-24 * time.Hour)

	entity, err := service.Create(context.Background(), threat.CreateParams{
		MD5:         "D41D8CD98F00B204E9800998ECF8427E",
		Name:        "Trojan.Win32.Generic",
		Type:        threat.TypeTrojan,
		Severity:    threat.SeverityHigh,
		Description: "Credential-stealing Trojan.",
		Categories:  []string{"banker", "stealer"},
		FirstSeenAt: &firstSeen,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	const expectedMD5 = threat.MD5("d41d8cd98f00b204e9800998ecf8427e")
	if entity.MD5 != expectedMD5 {
		t.Fatalf("MD5 = %q, want %q", entity.MD5, expectedMD5)
	}
	if !entity.CreatedAt.Equal(now) || !entity.UpdatedAt.Equal(now) {
		t.Fatalf("timestamps = (%v, %v), want both %v", entity.CreatedAt, entity.UpdatedAt, now)
	}
	if _, exists := repo.entities[expectedMD5]; !exists {
		t.Fatal("threat was not persisted")
	}
}

func TestThreatServiceCreateReturnsValidationErrorBeforeRepository(t *testing.T) {
	t.Parallel()

	repo := newMemoryThreatRepository()
	service := NewThreatService(repo, fixedClock{})

	_, err := service.Create(context.Background(), threat.CreateParams{
		MD5:      "invalid",
		Name:     "Trojan.Win32.Generic",
		Type:     threat.TypeTrojan,
		Severity: threat.SeverityHigh,
	})
	assertValidationError(t, err)
	if repo.createCalls != 0 {
		t.Fatalf("repository Create() was called %d times, want 0", repo.createCalls)
	}
}

func TestThreatServiceGetByMD5(t *testing.T) {
	t.Parallel()

	const md5 = threat.MD5("d41d8cd98f00b204e9800998ecf8427e")
	repo := newMemoryThreatRepository()
	repo.entities[md5] = existingThreat(md5)
	service := NewThreatService(repo, fixedClock{})

	entity, err := service.GetByMD5(context.Background(), "D41D8CD98F00B204E9800998ECF8427E")
	if err != nil {
		t.Fatalf("GetByMD5() error = %v", err)
	}
	if entity.MD5 != md5 {
		t.Fatalf("MD5 = %q, want %q", entity.MD5, md5)
	}

	_, err = service.GetByMD5(context.Background(), "invalid")
	assertValidationError(t, err)
}

func TestThreatServiceUpdate(t *testing.T) {
	t.Parallel()

	const md5 = threat.MD5("d41d8cd98f00b204e9800998ecf8427e")
	now := time.Date(2026, time.November, 3, 11, 0, 0, 0, time.UTC)
	repo := newMemoryThreatRepository()
	repo.entities[md5] = existingThreat(md5)
	service := NewThreatService(repo, fixedClock{now: now})
	params := threat.UpdateParams{
		Name:        "Trojan.Win32.Banker",
		Type:        threat.TypeTrojan,
		Severity:    threat.SeverityCritical,
		Description: "Updated classification.",
		Categories:  []string{"banker", "credential-theft"},
	}

	entity, err := service.Update(context.Background(), "D41D8CD98F00B204E9800998ECF8427E", params)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if entity.Name != params.Name || entity.Severity != threat.SeverityCritical {
		t.Fatalf("updated entity = %+v", entity)
	}
	if !entity.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %v, want %v", entity.UpdatedAt, now)
	}
	if entity.CreatedAt != repo.entities[md5].CreatedAt {
		t.Fatal("CreatedAt changed during update")
	}

	params.Categories[0] = "changed-by-caller"
	if repo.entities[md5].Categories[0] != "banker" {
		t.Fatal("repository state shares categories slice with caller")
	}
}

func TestThreatServiceUpdateReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := NewThreatService(newMemoryThreatRepository(), fixedClock{})
	_, err := service.Update(context.Background(), "d41d8cd98f00b204e9800998ecf8427e", validUpdateParams())
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, repository.ErrNotFound)
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type memoryThreatRepository struct {
	entities    map[threat.MD5]threat.Threat
	createCalls int
}

func newMemoryThreatRepository() *memoryThreatRepository {
	return &memoryThreatRepository{entities: make(map[threat.MD5]threat.Threat)}
}

func (r *memoryThreatRepository) Create(_ context.Context, entity threat.Threat) error {
	r.createCalls++
	if _, exists := r.entities[entity.MD5]; exists {
		return repository.ErrAlreadyExists
	}

	r.entities[entity.MD5] = entity
	return nil
}

func (r *memoryThreatRepository) GetByMD5(_ context.Context, md5 threat.MD5) (threat.Threat, error) {
	entity, exists := r.entities[md5]
	if !exists {
		return threat.Threat{}, repository.ErrNotFound
	}

	return entity, nil
}

func (r *memoryThreatRepository) Update(_ context.Context, entity threat.Threat) error {
	if _, exists := r.entities[entity.MD5]; !exists {
		return repository.ErrNotFound
	}

	r.entities[entity.MD5] = entity
	return nil
}

func existingThreat(md5 threat.MD5) threat.Threat {
	createdAt := time.Date(2026, time.November, 1, 10, 0, 0, 0, time.UTC)
	return threat.Threat{
		MD5:        md5,
		Name:       "Trojan.Win32.Generic",
		Type:       threat.TypeTrojan,
		Severity:   threat.SeverityHigh,
		Categories: []string{"banker"},
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
}

func validUpdateParams() threat.UpdateParams {
	return threat.UpdateParams{
		Name:     "Trojan.Win32.Generic",
		Type:     threat.TypeTrojan,
		Severity: threat.SeverityHigh,
	}
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()

	var validationError *threat.ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("error = %v, want validation error", err)
	}
}
