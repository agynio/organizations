package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const organizationColumns = `id, name, slug, sandbox_default_idle_timeout, sandbox_default_ttl, created_at, updated_at`

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func scanOrganization(row pgx.Row) (Organization, error) {
	var organization Organization
	if err := row.Scan(
		&organization.ID,
		&organization.Name,
		&organization.Slug,
		&organization.SandboxDefaultIdleTimeout,
		&organization.SandboxDefaultTTL,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	); err != nil {
		return Organization{}, err
	}
	return organization, nil
}

func (s *Store) CreateOrganization(ctx context.Context, input OrganizationInput) (Organization, error) {
	row := s.pool.QueryRow(ctx,
		fmt.Sprintf(`INSERT INTO organizations (name, slug)
         VALUES ($1, $2)
         RETURNING %s`, organizationColumns),
		input.Name, input.Slug,
	)
	organization, err := scanOrganization(row)
	if err != nil {
		if isUniqueViolation(err) {
			return Organization{}, AlreadyExists("organization slug")
		}
		return Organization{}, err
	}
	return organization, nil
}

func (s *Store) GetOrganization(ctx context.Context, id uuid.UUID) (Organization, error) {
	row := s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM organizations WHERE id = $1`, organizationColumns),
		id,
	)
	organization, err := scanOrganization(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Organization{}, NotFound("organization")
		}
		return Organization{}, err
	}
	return organization, nil
}

// GetOrganizationBySlug resolves the slug that appears in an app address and in
// an image proxy reference.
func (s *Store) GetOrganizationBySlug(ctx context.Context, slug string) (Organization, error) {
	row := s.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM organizations WHERE slug = $1`, organizationColumns),
		slug,
	)
	organization, err := scanOrganization(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Organization{}, NotFound("organization")
		}
		return Organization{}, err
	}
	return organization, nil
}

func (s *Store) UpdateOrganization(ctx context.Context, id uuid.UUID, update OrganizationUpdate) (Organization, error) {
	builder := updateBuilder{}
	if update.Name != nil {
		builder.add("name", *update.Name)
	}
	if update.Slug != nil {
		builder.add("slug", *update.Slug)
	}
	if update.SandboxDefaultIdleTimeout != nil {
		builder.add("sandbox_default_idle_timeout", *update.SandboxDefaultIdleTimeout)
	}
	if update.SandboxDefaultTTL != nil {
		builder.add("sandbox_default_ttl", *update.SandboxDefaultTTL)
	}

	if builder.empty() {
		return Organization{}, fmt.Errorf("organization update requires at least one field")
	}
	query, args := builder.build("organizations", organizationColumns, id)
	row := s.pool.QueryRow(ctx, query, args...)
	organization, err := scanOrganization(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Organization{}, NotFound("organization")
		}
		if isUniqueViolation(err) {
			return Organization{}, AlreadyExists("organization slug")
		}
		return Organization{}, err
	}
	return organization, nil
}

func (s *Store) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return NotFound("organization")
	}
	return nil
}

func (s *Store) ListOrganizations(ctx context.Context, _ OrganizationFilter, pageSize int32, cursor *PageCursor) (OrganizationListResult, error) {
	organizations, nextCursor, err := listEntities(ctx, s.pool,
		fmt.Sprintf("SELECT %s FROM organizations", organizationColumns),
		nil,
		nil,
		cursor,
		pageSize,
		scanOrganization,
		func(organization Organization) uuid.UUID { return organization.ID },
	)
	if err != nil {
		return OrganizationListResult{}, err
	}
	return OrganizationListResult{Organizations: organizations, NextCursor: nextCursor}, nil
}

func (s *Store) GetOrganizationsByIDs(ctx context.Context, ids []uuid.UUID) ([]Organization, error) {
	if len(ids) == 0 {
		return []Organization{}, nil
	}
	idArray := pgtype.FlatArray[uuid.UUID](ids)
	rows, err := s.pool.Query(ctx,
		fmt.Sprintf("SELECT %s FROM organizations WHERE id = ANY($1) ORDER BY id ASC", organizationColumns),
		idArray,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	organizations := make([]Organization, 0, len(ids))
	for rows.Next() {
		organization, err := scanOrganization(rows)
		if err != nil {
			return nil, err
		}
		organizations = append(organizations, organization)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return organizations, nil
}
