package store

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sertdev/pxbin/internal/crypto"
)

type Upstream struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	BaseURL         string    `json:"base_url"`
	APIKeyEncrypted string    `json:"-"` // never expose in JSON
	Format          string    `json:"format"`
	IsActive        bool      `json:"is_active"`
	Priority        int       `json:"priority"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UpstreamCreate struct {
	Name     string `json:"name"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Format   string `json:"format"`
	Priority int    `json:"priority"`
}

type UpstreamUpdate struct {
	Name     *string `json:"name,omitempty"`
	BaseURL  *string `json:"base_url,omitempty"`
	APIKey   *string `json:"api_key,omitempty"`
	Format   *string `json:"format,omitempty"`
	Priority *int    `json:"priority,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// encryptAPIKey encrypts an API key if an encryption key is configured.
func (s *Store) encryptAPIKey(apiKey string) string {
	if s.encryptionKey == nil || apiKey == "" {
		return apiKey
	}
	encrypted, err := crypto.Encrypt([]byte(apiKey), s.encryptionKey)
	if err != nil {
		log.Printf("warning: failed to encrypt api key: %v", err)
		return apiKey
	}
	return encrypted
}

// decryptAPIKey decrypts an API key if it's encrypted. Handles legacy
// plaintext values gracefully.
func (s *Store) decryptAPIKey(stored string) string {
	if s.encryptionKey == nil || stored == "" {
		return stored
	}
	if !crypto.IsEncrypted(stored) {
		log.Printf("warning: upstream api key is not encrypted (legacy plaintext)")
		return stored
	}
	decrypted, err := crypto.Decrypt(stored, s.encryptionKey)
	if err != nil {
		log.Printf("warning: failed to decrypt api key: %v", err)
		return stored
	}
	return string(decrypted)
}

func (s *Store) ListUpstreams(ctx context.Context) ([]Upstream, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, base_url, api_key_encrypted, format, is_active, priority, created_at, updated_at
		FROM upstreams ORDER BY priority DESC, name
	`)
	if err != nil {
		return nil, fmt.Errorf("list upstreams: %w", err)
	}
	defer rows.Close()

	var upstreams []Upstream
	for rows.Next() {
		var u Upstream
		if err := rows.Scan(
			&u.ID, &u.Name, &u.BaseURL, &u.APIKeyEncrypted,
			&u.Format, &u.IsActive, &u.Priority, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan upstream: %w", err)
		}
		u.APIKeyEncrypted = s.decryptAPIKey(u.APIKeyEncrypted)
		upstreams = append(upstreams, u)
	}
	return upstreams, rows.Err()
}

func (s *Store) GetUpstream(ctx context.Context, id uuid.UUID) (*Upstream, error) {
	var u Upstream
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, base_url, api_key_encrypted, format, is_active, priority, created_at, updated_at
		FROM upstreams WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Name, &u.BaseURL, &u.APIKeyEncrypted,
		&u.Format, &u.IsActive, &u.Priority, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get upstream: %w", err)
	}
	u.APIKeyEncrypted = s.decryptAPIKey(u.APIKeyEncrypted)
	return &u, nil
}

func (s *Store) GetActiveUpstream(ctx context.Context) (*Upstream, error) {
	var u Upstream
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, base_url, api_key_encrypted, format, is_active, priority, created_at, updated_at
		FROM upstreams WHERE is_active = true ORDER BY priority DESC LIMIT 1
	`).Scan(
		&u.ID, &u.Name, &u.BaseURL, &u.APIKeyEncrypted,
		&u.Format, &u.IsActive, &u.Priority, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active upstream: %w", err)
	}
	u.APIKeyEncrypted = s.decryptAPIKey(u.APIKeyEncrypted)
	return &u, nil
}

func (s *Store) CreateUpstream(ctx context.Context, uc *UpstreamCreate) (*Upstream, error) {
	format := uc.Format
	if format == "" {
		format = "openai"
	}
	encryptedKey := s.encryptAPIKey(uc.APIKey)
	var u Upstream
	err := s.pool.QueryRow(ctx, `
		INSERT INTO upstreams (name, base_url, api_key_encrypted, format, priority)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, base_url, api_key_encrypted, format, is_active, priority, created_at, updated_at
	`, uc.Name, uc.BaseURL, encryptedKey, format, uc.Priority).Scan(
		&u.ID, &u.Name, &u.BaseURL, &u.APIKeyEncrypted,
		&u.Format, &u.IsActive, &u.Priority, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create upstream: %w", err)
	}
	u.APIKeyEncrypted = s.decryptAPIKey(u.APIKeyEncrypted)
	return &u, nil
}

func (s *Store) UpdateUpstream(ctx context.Context, id uuid.UUID, upd *UpstreamUpdate) error {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if upd.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *upd.Name)
		argIdx++
	}
	if upd.BaseURL != nil {
		sets = append(sets, fmt.Sprintf("base_url = $%d", argIdx))
		args = append(args, *upd.BaseURL)
		argIdx++
	}
	if upd.APIKey != nil {
		sets = append(sets, fmt.Sprintf("api_key_encrypted = $%d", argIdx))
		args = append(args, s.encryptAPIKey(*upd.APIKey))
		argIdx++
	}
	if upd.Format != nil {
		sets = append(sets, fmt.Sprintf("format = $%d", argIdx))
		args = append(args, *upd.Format)
		argIdx++
	}
	if upd.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *upd.Priority)
		argIdx++
	}
	if upd.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *upd.IsActive)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = now()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE upstreams SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update upstream: %w", err)
	}
	return nil
}

func (s *Store) DeleteUpstream(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clear FK references before deleting.
	if _, err := tx.Exec(ctx, "UPDATE models SET upstream_id = NULL WHERE upstream_id = $1", id); err != nil {
		return fmt.Errorf("clear model refs: %w", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE request_logs SET upstream_id = NULL WHERE upstream_id = $1", id); err != nil {
		return fmt.Errorf("clear log refs: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM upstreams WHERE id = $1", id); err != nil {
		return fmt.Errorf("delete upstream: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *Store) DeleteUpstreams(ctx context.Context, ids []uuid.UUID) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "UPDATE models SET upstream_id = NULL WHERE upstream_id = ANY($1)", ids); err != nil {
		return 0, fmt.Errorf("clear model refs: %w", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE request_logs SET upstream_id = NULL WHERE upstream_id = ANY($1)", ids); err != nil {
		return 0, fmt.Errorf("clear log refs: %w", err)
	}
	ct, err := tx.Exec(ctx, "DELETE FROM upstreams WHERE id = ANY($1)", ids)
	if err != nil {
		return 0, fmt.Errorf("delete upstreams: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return ct.RowsAffected(), nil
}
