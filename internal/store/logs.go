package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LogEntry struct {
	KeyID              uuid.UUID
	Timestamp          time.Time
	Method             string
	Path               string
	Model              string
	InputFormat        string // "anthropic" or "openai"
	UpstreamID         *uuid.UUID
	StatusCode         int
	LatencyMS          int
	InputTokens        int
	OutputTokens       int
	CacheCreationTokens int
	CacheReadTokens    int
	Cost               float64
	OverheadUS         int
	ErrorMessage       string
	RequestMetadata    map[string]interface{}
}

type RequestLog struct {
	ID              uuid.UUID              `json:"id"`
	KeyID           *uuid.UUID             `json:"llm_key_id"`
	Timestamp       time.Time              `json:"timestamp"`
	Method          string                 `json:"method"`
	Path            string                 `json:"path"`
	Model           *string                `json:"model"`
	InputFormat     string                 `json:"input_format"`
	UpstreamID      *uuid.UUID             `json:"upstream_id"`
	StatusCode      *int                   `json:"status_code"`
	LatencyMS       *int                   `json:"latency_ms"`
	InputTokens     *int                   `json:"input_tokens"`
	OutputTokens    *int                   `json:"output_tokens"`
	Cost            *float64               `json:"cost"`
	OverheadUS      *int                   `json:"overhead_us"`
	ErrorMessage    *string                `json:"error_message"`
	RequestMetadata map[string]interface{} `json:"request_metadata"`
	CreatedAt       time.Time              `json:"created_at"`
}

type LogFilter struct {
	KeyID       *uuid.UUID
	Model       *string
	StatusCode  *int
	InputFormat *string
	DateFrom    *time.Time
	DateTo      *time.Time
	Page        int
	PerPage     int
}

func (s *Store) InsertLog(ctx context.Context, entry *LogEntry) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO request_logs (
			llm_key_id, timestamp, method, path, model, input_format,
			upstream_id, status_code, latency_ms, input_tokens, output_tokens,
			cache_creation_tokens, cache_read_tokens, cost, overhead_us, error_message, request_metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		entry.KeyID, entry.Timestamp, entry.Method, entry.Path, entry.Model, entry.InputFormat,
		entry.UpstreamID, entry.StatusCode, entry.LatencyMS, entry.InputTokens, entry.OutputTokens,
		entry.CacheCreationTokens, entry.CacheReadTokens, entry.Cost, entry.OverheadUS, entry.ErrorMessage, entry.RequestMetadata,
	)
	if err != nil {
		return fmt.Errorf("insert log: %w", err)
	}
	return nil
}

func (s *Store) InsertLogBatch(ctx context.Context, entries []*LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO request_logs (
			llm_key_id, timestamp, method, path, model, input_format,
			upstream_id, status_code, latency_ms, input_tokens, output_tokens,
			cache_creation_tokens, cache_read_tokens, cost, overhead_us, error_message, request_metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	for _, entry := range entries {
		batch.Queue(query,
			entry.KeyID, entry.Timestamp, entry.Method, entry.Path, entry.Model, entry.InputFormat,
			entry.UpstreamID, entry.StatusCode, entry.LatencyMS, entry.InputTokens, entry.OutputTokens,
			entry.CacheCreationTokens, entry.CacheReadTokens, entry.Cost, entry.OverheadUS, entry.ErrorMessage, entry.RequestMetadata,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range entries {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert log batch: %w", err)
		}
	}
	return nil
}

func (s *Store) GetLog(ctx context.Context, id uuid.UUID) (*RequestLog, error) {
	var log RequestLog
	err := s.pool.QueryRow(ctx, `
		SELECT id, llm_key_id, timestamp, method, path, model, input_format,
		       upstream_id, status_code, latency_ms, input_tokens, output_tokens,
		       cost, overhead_us, error_message, request_metadata, created_at
		FROM request_logs WHERE id = $1
	`, id).Scan(
		&log.ID, &log.KeyID, &log.Timestamp, &log.Method, &log.Path, &log.Model, &log.InputFormat,
		&log.UpstreamID, &log.StatusCode, &log.LatencyMS, &log.InputTokens, &log.OutputTokens,
		&log.Cost, &log.OverheadUS, &log.ErrorMessage, &log.RequestMetadata, &log.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get log: %w", err)
	}
	return &log, nil
}

func (s *Store) ListLogs(ctx context.Context, filter LogFilter) ([]RequestLog, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.KeyID != nil {
		conditions = append(conditions, fmt.Sprintf("llm_key_id = $%d", argIdx))
		args = append(args, *filter.KeyID)
		argIdx++
	}
	if filter.Model != nil {
		conditions = append(conditions, fmt.Sprintf("model ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, *filter.Model)
		argIdx++
	}
	if filter.StatusCode != nil {
		if *filter.StatusCode%100 == 0 {
			conditions = append(conditions, fmt.Sprintf("status_code >= $%d AND status_code < $%d", argIdx, argIdx+1))
			args = append(args, *filter.StatusCode, *filter.StatusCode+100)
			argIdx += 2
		} else {
			conditions = append(conditions, fmt.Sprintf("status_code = $%d", argIdx))
			args = append(args, *filter.StatusCode)
			argIdx++
		}
	}
	if filter.InputFormat != nil {
		conditions = append(conditions, fmt.Sprintf("input_format = $%d", argIdx))
		args = append(args, *filter.InputFormat)
		argIdx++
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *filter.DateTo)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT id, llm_key_id, timestamp, method, path, model, input_format,
		       upstream_id, status_code, latency_ms, input_tokens, output_tokens,
		       cost, overhead_us, error_message, request_metadata, created_at,
		       COUNT(*) OVER() as total
		FROM request_logs %s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list logs: %w", err)
	}
	defer rows.Close()

	var logs []RequestLog
	var total int
	for rows.Next() {
		var log RequestLog
		if err := rows.Scan(
			&log.ID, &log.KeyID, &log.Timestamp, &log.Method, &log.Path, &log.Model, &log.InputFormat,
			&log.UpstreamID, &log.StatusCode, &log.LatencyMS, &log.InputTokens, &log.OutputTokens,
			&log.Cost, &log.OverheadUS, &log.ErrorMessage, &log.RequestMetadata, &log.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan log: %w", err)
		}
		logs = append(logs, log)
	}
	return logs, total, rows.Err()
}

func (s *Store) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	ct, err := s.pool.Exec(ctx, "DELETE FROM request_logs WHERE timestamp < $1", olderThan)
	if err != nil {
		return 0, fmt.Errorf("delete old logs: %w", err)
	}
	return ct.RowsAffected(), nil
}
