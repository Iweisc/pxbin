package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LLMAPIKey struct {
	ID         uuid.UUID        `json:"id"`
	KeyHash    string           `json:"-"`
	KeyPrefix  string           `json:"key_prefix"`
	Name       string           `json:"name"`
	IsActive   bool             `json:"is_active"`
	RateLimit  *int             `json:"rate_limit"`
	LastUsedAt *time.Time       `json:"last_used_at"`
	Metadata   json.RawMessage  `json:"metadata"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

type ManagementAPIKey struct {
	ID          uuid.UUID  `json:"id"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"key_prefix"`
	Name        string     `json:"name"`
	IsActive    bool       `json:"is_active"`
	Permissions []string   `json:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type LLMKeyUpdate struct {
	Name      *string `json:"name"`
	IsActive  *bool   `json:"is_active"`
	RateLimit *int    `json:"rate_limit"`
}

type ManagementKeyUpdate struct {
	Name        *string  `json:"name"`
	IsActive    *bool    `json:"is_active"`
	Permissions []string `json:"permissions"`
}

func (s *Store) GetLLMKeyByHash(ctx context.Context, hash string) (*LLMAPIKey, error) {
	var k LLMAPIKey
	err := s.pool.QueryRow(ctx, `
		SELECT id, key_hash, key_prefix, name, is_active, rate_limit, last_used_at, metadata, created_at, updated_at
		FROM llm_api_keys WHERE key_hash = $1
	`, hash).Scan(
		&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.IsActive,
		&k.RateLimit, &k.LastUsedAt, &k.Metadata, &k.CreatedAt, &k.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get llm key by hash: %w", err)
	}
	return &k, nil
}

func (s *Store) ListLLMKeys(ctx context.Context, page, perPage int) ([]LLMAPIKey, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM llm_api_keys").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count llm keys: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.pool.Query(ctx, `
		SELECT id, key_prefix, name, is_active, rate_limit, last_used_at, metadata, created_at, updated_at
		FROM llm_api_keys ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list llm keys: %w", err)
	}
	defer rows.Close()

	var keys []LLMAPIKey
	for rows.Next() {
		var k LLMAPIKey
		if err := rows.Scan(
			&k.ID, &k.KeyPrefix, &k.Name, &k.IsActive,
			&k.RateLimit, &k.LastUsedAt, &k.Metadata, &k.CreatedAt, &k.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan llm key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, total, rows.Err()
}

func (s *Store) CreateLLMKey(ctx context.Context, keyHash, keyPrefix, name string, rateLimit *int) (*LLMAPIKey, error) {
	var k LLMAPIKey
	err := s.pool.QueryRow(ctx, `
		INSERT INTO llm_api_keys (key_hash, key_prefix, name, rate_limit)
		VALUES ($1, $2, $3, $4)
		RETURNING id, key_hash, key_prefix, name, is_active, rate_limit, last_used_at, metadata, created_at, updated_at
	`, keyHash, keyPrefix, name, rateLimit).Scan(
		&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.IsActive,
		&k.RateLimit, &k.LastUsedAt, &k.Metadata, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create llm key: %w", err)
	}
	return &k, nil
}

func (s *Store) UpdateLLMKey(ctx context.Context, id uuid.UUID, updates LLMKeyUpdate) error {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if updates.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *updates.Name)
		argIdx++
	}
	if updates.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *updates.IsActive)
		argIdx++
	}
	if updates.RateLimit != nil {
		sets = append(sets, fmt.Sprintf("rate_limit = $%d", argIdx))
		args = append(args, *updates.RateLimit)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE llm_api_keys SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update llm key: %w", err)
	}
	return nil
}

func (s *Store) DeactivateLLMKey(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		"UPDATE llm_api_keys SET is_active = false, updated_at = now() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deactivate llm key: %w", err)
	}
	return nil
}

func (s *Store) UpdateLLMKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		"UPDATE llm_api_keys SET last_used_at = now() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("update llm key last used: %w", err)
	}
	return nil
}

func (s *Store) BatchUpdateLLMKeyLastUsed(ctx context.Context, ids []uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		"UPDATE llm_api_keys SET last_used_at = now() WHERE id = ANY($1)", ids)
	if err != nil {
		return fmt.Errorf("batch update llm key last used: %w", err)
	}
	return nil
}

func (s *Store) GetManagementKeyByHash(ctx context.Context, hash string) (*ManagementAPIKey, error) {
	var k ManagementAPIKey
	err := s.pool.QueryRow(ctx, `
		SELECT id, key_hash, key_prefix, name, is_active, permissions, last_used_at, created_at, updated_at
		FROM management_api_keys WHERE key_hash = $1
	`, hash).Scan(
		&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.IsActive,
		&k.Permissions, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get management key by hash: %w", err)
	}
	return &k, nil
}

func (s *Store) ListManagementKeys(ctx context.Context, page, perPage int) ([]ManagementAPIKey, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM management_api_keys").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count management keys: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.pool.Query(ctx, `
		SELECT id, key_prefix, name, is_active, permissions, last_used_at, created_at, updated_at
		FROM management_api_keys ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list management keys: %w", err)
	}
	defer rows.Close()

	var keys []ManagementAPIKey
	for rows.Next() {
		var k ManagementAPIKey
		if err := rows.Scan(
			&k.ID, &k.KeyPrefix, &k.Name, &k.IsActive,
			&k.Permissions, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan management key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, total, rows.Err()
}

func (s *Store) CreateManagementKey(ctx context.Context, keyHash, keyPrefix, name string, permissions []string) (*ManagementAPIKey, error) {
	var k ManagementAPIKey
	err := s.pool.QueryRow(ctx, `
		INSERT INTO management_api_keys (key_hash, key_prefix, name, permissions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, key_hash, key_prefix, name, is_active, permissions, last_used_at, created_at, updated_at
	`, keyHash, keyPrefix, name, permissions).Scan(
		&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.IsActive,
		&k.Permissions, &k.LastUsedAt, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create management key: %w", err)
	}
	return &k, nil
}

func (s *Store) UpdateManagementKey(ctx context.Context, id uuid.UUID, updates ManagementKeyUpdate) error {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if updates.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *updates.Name)
		argIdx++
	}
	if updates.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *updates.IsActive)
		argIdx++
	}
	if updates.Permissions != nil {
		sets = append(sets, fmt.Sprintf("permissions = $%d", argIdx))
		args = append(args, updates.Permissions)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE management_api_keys SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update management key: %w", err)
	}
	return nil
}

func (s *Store) DeactivateManagementKey(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		"UPDATE management_api_keys SET is_active = false, updated_at = now() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deactivate management key: %w", err)
	}
	return nil
}

