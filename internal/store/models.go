package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Model struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	DisplayName          *string    `json:"display_name"`
	Provider             string     `json:"provider"`
	UpstreamID           *uuid.UUID `json:"upstream_id"`
	InputCostPerMillion  float64    `json:"input_cost_per_million"`
	OutputCostPerMillion float64    `json:"output_cost_per_million"`
	IsActive             bool       `json:"is_active"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type ModelWithUpstream struct {
	Model
	UpstreamBaseURL string
	UpstreamAPIKey  string
	UpstreamFormat  string
}

type ModelCreate struct {
	Name                 string     `json:"name"`
	DisplayName          *string    `json:"display_name"`
	Provider             string     `json:"provider"`
	UpstreamID           *uuid.UUID `json:"upstream_id"`
	InputCostPerMillion  float64    `json:"input_cost_per_million"`
	OutputCostPerMillion float64    `json:"output_cost_per_million"`
}

type ModelUpdate struct {
	Name                 *string    `json:"name,omitempty"`
	DisplayName          *string    `json:"display_name,omitempty"`
	Provider             *string    `json:"provider,omitempty"`
	UpstreamID           *uuid.UUID `json:"upstream_id,omitempty"`
	InputCostPerMillion  *float64   `json:"input_cost_per_million,omitempty"`
	OutputCostPerMillion *float64   `json:"output_cost_per_million,omitempty"`
	IsActive             *bool      `json:"is_active,omitempty"`
}

func (s *Store) ListModels(ctx context.Context) ([]Model, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, display_name, provider, upstream_id, input_cost_per_million, output_cost_per_million, is_active, created_at, updated_at
		FROM models ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer rows.Close()

	var models []Model
	for rows.Next() {
		var m Model
		if err := rows.Scan(
			&m.ID, &m.Name, &m.DisplayName, &m.Provider, &m.UpstreamID,
			&m.InputCostPerMillion, &m.OutputCostPerMillion,
			&m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

func (s *Store) GetModel(ctx context.Context, id uuid.UUID) (*Model, error) {
	var m Model
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, display_name, provider, upstream_id, input_cost_per_million, output_cost_per_million, is_active, created_at, updated_at
		FROM models WHERE id = $1
	`, id).Scan(
		&m.ID, &m.Name, &m.DisplayName, &m.Provider, &m.UpstreamID,
		&m.InputCostPerMillion, &m.OutputCostPerMillion,
		&m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model: %w", err)
	}
	return &m, nil
}

func (s *Store) GetModelByName(ctx context.Context, name string) (*Model, error) {
	var m Model
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, display_name, provider, upstream_id, input_cost_per_million, output_cost_per_million, is_active, created_at, updated_at
		FROM models WHERE name = $1
	`, name).Scan(
		&m.ID, &m.Name, &m.DisplayName, &m.Provider, &m.UpstreamID,
		&m.InputCostPerMillion, &m.OutputCostPerMillion,
		&m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model by name: %w", err)
	}
	return &m, nil
}

func (s *Store) CreateModel(ctx context.Context, mc *ModelCreate) (*Model, error) {
	var m Model
	err := s.pool.QueryRow(ctx, `
		INSERT INTO models (name, display_name, provider, upstream_id, input_cost_per_million, output_cost_per_million)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, display_name, provider, upstream_id, input_cost_per_million, output_cost_per_million, is_active, created_at, updated_at
	`, mc.Name, mc.DisplayName, mc.Provider, mc.UpstreamID, mc.InputCostPerMillion, mc.OutputCostPerMillion).Scan(
		&m.ID, &m.Name, &m.DisplayName, &m.Provider, &m.UpstreamID,
		&m.InputCostPerMillion, &m.OutputCostPerMillion,
		&m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create model: %w", err)
	}
	return &m, nil
}

func (s *Store) UpdateModel(ctx context.Context, id uuid.UUID, u *ModelUpdate) error {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if u.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *u.Name)
		argIdx++
	}
	if u.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *u.DisplayName)
		argIdx++
	}
	if u.Provider != nil {
		sets = append(sets, fmt.Sprintf("provider = $%d", argIdx))
		args = append(args, *u.Provider)
		argIdx++
	}
	if u.UpstreamID != nil {
		sets = append(sets, fmt.Sprintf("upstream_id = $%d", argIdx))
		args = append(args, *u.UpstreamID)
		argIdx++
	}
	if u.InputCostPerMillion != nil {
		sets = append(sets, fmt.Sprintf("input_cost_per_million = $%d", argIdx))
		args = append(args, *u.InputCostPerMillion)
		argIdx++
	}
	if u.OutputCostPerMillion != nil {
		sets = append(sets, fmt.Sprintf("output_cost_per_million = $%d", argIdx))
		args = append(args, *u.OutputCostPerMillion)
		argIdx++
	}
	if u.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *u.IsActive)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE models SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update model: %w", err)
	}
	return nil
}

func (s *Store) DeleteModel(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM models WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete model: %w", err)
	}
	return nil
}

func (s *Store) DeleteModels(ctx context.Context, ids []uuid.UUID) (int64, error) {
	ct, err := s.pool.Exec(ctx, "DELETE FROM models WHERE id = ANY($1)", ids)
	if err != nil {
		return 0, fmt.Errorf("delete models: %w", err)
	}
	return ct.RowsAffected(), nil
}

// GetModelWithUpstream joins models with their linked upstream in a single
// query. Returns nil if the model doesn't exist or has no linked upstream.
func (s *Store) GetModelWithUpstream(ctx context.Context, modelName string) (*ModelWithUpstream, error) {
	var mw ModelWithUpstream
	err := s.pool.QueryRow(ctx, `
		SELECT m.id, m.name, m.display_name, m.provider, m.upstream_id,
		       m.input_cost_per_million, m.output_cost_per_million,
		       m.is_active, m.created_at, m.updated_at,
		       u.base_url, u.api_key_encrypted, u.format
		FROM models m
		JOIN upstreams u ON u.id = m.upstream_id
		WHERE m.name = $1 AND m.is_active = true AND u.is_active = true
	`, modelName).Scan(
		&mw.ID, &mw.Name, &mw.DisplayName, &mw.Provider, &mw.UpstreamID,
		&mw.InputCostPerMillion, &mw.OutputCostPerMillion,
		&mw.IsActive, &mw.CreatedAt, &mw.UpdatedAt,
		&mw.UpstreamBaseURL, &mw.UpstreamAPIKey, &mw.UpstreamFormat,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model with upstream: %w", err)
	}
	mw.UpstreamAPIKey = s.decryptAPIKey(mw.UpstreamAPIKey)
	return &mw, nil
}

// ListActiveModelsWithUpstream returns all active models joined with their
// active upstream configuration.
func (s *Store) ListActiveModelsWithUpstream(ctx context.Context) ([]*ModelWithUpstream, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.name, m.display_name, m.provider, m.upstream_id,
		       m.input_cost_per_million, m.output_cost_per_million,
		       m.is_active, m.created_at, m.updated_at,
		       u.base_url, u.api_key_encrypted, u.format
		FROM models m
		JOIN upstreams u ON u.id = m.upstream_id
		WHERE m.is_active = true AND u.is_active = true
	`)
	if err != nil {
		return nil, fmt.Errorf("list active models with upstream: %w", err)
	}
	defer rows.Close()

	models := make([]*ModelWithUpstream, 0)
	for rows.Next() {
		var mw ModelWithUpstream
		if err := rows.Scan(
			&mw.ID, &mw.Name, &mw.DisplayName, &mw.Provider, &mw.UpstreamID,
			&mw.InputCostPerMillion, &mw.OutputCostPerMillion,
			&mw.IsActive, &mw.CreatedAt, &mw.UpdatedAt,
			&mw.UpstreamBaseURL, &mw.UpstreamAPIKey, &mw.UpstreamFormat,
		); err != nil {
			return nil, fmt.Errorf("scan active model with upstream: %w", err)
		}
		mw.UpstreamAPIKey = s.decryptAPIKey(mw.UpstreamAPIKey)
		models = append(models, &mw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active models with upstream: %w", err)
	}
	return models, nil
}
