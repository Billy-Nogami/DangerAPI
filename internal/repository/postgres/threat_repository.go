package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"threat-service/internal/domain/threat"
	"threat-service/internal/repository"
)

// ThreatRepository сохраняет угрозы в PostgreSQL.
type ThreatRepository struct {
	pool *pgxpool.Pool
}

// NewThreatRepository создаёт репозиторий поверх пула соединений PostgreSQL.
func NewThreatRepository(pool *pgxpool.Pool) *ThreatRepository {
	return &ThreatRepository{pool: pool}
}

// Create сохраняет новую угрозу и сообщает о дублирующемся MD5.
func (r *ThreatRepository) Create(ctx context.Context, entity threat.Threat) error {
	const query = `
		INSERT INTO threats (
			md5, name, type, severity, description, categories,
			first_seen_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(
		ctx,
		query,
		string(entity.MD5),
		entity.Name,
		string(entity.Type),
		string(entity.Severity),
		entity.Description,
		categoriesForStorage(entity.Categories),
		entity.FirstSeenAt,
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	if err != nil {
		return mapDatabaseError(err)
	}

	return nil
}

// GetByMD5 возвращает угрозу по её MD5-хешу.
func (r *ThreatRepository) GetByMD5(ctx context.Context, md5 threat.MD5) (threat.Threat, error) {
	const query = `
		SELECT
			md5, name, type, severity, description, categories,
			first_seen_at, created_at, updated_at
		FROM threats
		WHERE md5 = $1`

	row := r.pool.QueryRow(ctx, query, string(md5))
	entity, err := scanThreat(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return threat.Threat{}, repository.ErrNotFound
	}
	if err != nil {
		return threat.Threat{}, mapDatabaseError(err)
	}

	return entity, nil
}

// Update сохраняет все изменяемые поля существующей угрозы.
func (r *ThreatRepository) Update(ctx context.Context, entity threat.Threat) error {
	const query = `
		UPDATE threats
		SET
			name = $2,
			type = $3,
			severity = $4,
			description = $5,
			categories = $6,
			first_seen_at = $7,
			updated_at = $8
		WHERE md5 = $1`

	result, err := r.pool.Exec(
		ctx,
		query,
		string(entity.MD5),
		entity.Name,
		string(entity.Type),
		string(entity.Severity),
		entity.Description,
		categoriesForStorage(entity.Categories),
		entity.FirstSeenAt,
		entity.UpdatedAt,
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanThreat(row rowScanner) (threat.Threat, error) {
	var (
		entity      threat.Threat
		md5         string
		threatType  string
		severity    string
		firstSeenAt *time.Time
	)

	err := row.Scan(
		&md5,
		&entity.Name,
		&threatType,
		&severity,
		&entity.Description,
		&entity.Categories,
		&firstSeenAt,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		return threat.Threat{}, err
	}

	entity.MD5 = threat.MD5(md5)
	entity.Type = threat.Type(threatType)
	entity.Severity = threat.Severity(severity)
	entity.FirstSeenAt = firstSeenAt
	return entity, nil
}

func categoriesForStorage(categories []string) []string {
	if categories == nil {
		return []string{}
	}

	return categories
}

func mapDatabaseError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		return repository.ErrAlreadyExists
	}

	return fmt.Errorf("ошибка PostgreSQL: %w", err)
}
