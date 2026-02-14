package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OverviewStats struct {
	TotalRequests        int     `json:"total_requests"`
	TotalInputTokens     int64   `json:"total_input_tokens"`
	TotalOutputTokens    int64   `json:"total_output_tokens"`
	TotalCacheReadTokens int64   `json:"total_cache_read_tokens"`
	CacheHitRate         float64 `json:"cache_hit_rate"`
	TotalCost            float64 `json:"total_cost"`
	AvgLatencyMS         int     `json:"avg_latency_ms"`
	AvgOverheadUS        int     `json:"avg_overhead_us"`
	ErrorCount           int     `json:"error_count"`
	ErrorRate            float64 `json:"error_rate"`
}

type KeyStats struct {
	KeyID             uuid.UUID `json:"key_id"`
	KeyPrefix         string    `json:"key_prefix"`
	KeyName           string    `json:"key_name"`
	TotalRequests     int       `json:"total_requests"`
	TotalInputTokens  int64     `json:"total_input_tokens"`
	TotalOutputTokens int64     `json:"total_output_tokens"`
	TotalCost         float64   `json:"total_cost"`
	AvgLatencyMS      int       `json:"avg_latency_ms"`
}

type ModelStats struct {
	Model             string  `json:"model"`
	TotalRequests     int     `json:"total_requests"`
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalCost         float64 `json:"total_cost"`
	AvgLatencyMS      int     `json:"avg_latency_ms"`
}

type TimeSeriesBucket struct {
	Bucket        time.Time `json:"bucket"`
	Requests      int       `json:"requests"`
	InputTokens   int64     `json:"input_tokens"`
	OutputTokens  int64     `json:"output_tokens"`
	Cost          float64   `json:"cost"`
	AvgLatencyMS  int       `json:"avg_latency_ms"`
	AvgOverheadUS int       `json:"avg_overhead_us"`
	Errors        int       `json:"errors"`
}

type LatencyStats struct {
	P50         int `json:"p50"`
	P95         int `json:"p95"`
	P99         int `json:"p99"`
	OverheadP50US int `json:"overhead_p50_us"`
	OverheadP95US int `json:"overhead_p95_us"`
	OverheadP99US int `json:"overhead_p99_us"`
}

func periodToInterval(period string) string {
	switch period {
	case "24h":
		return "24 hours"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	default:
		return "24 hours"
	}
}

func intervalToTrunc(interval string) string {
	switch interval {
	case "1h":
		return "hour"
	case "1d":
		return "day"
	default:
		return "hour"
	}
}

func (s *Store) GetOverviewStats(ctx context.Context, period string) (*OverviewStats, error) {
	interval := periodToInterval(period)
	var stats OverviewStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(AVG(latency_ms)::int, 0) as avg_latency_ms,
			COALESCE(AVG(overhead_us)::int, 0) as avg_overhead_us,
			COUNT(*) FILTER (WHERE status_code >= 400) as error_count
		FROM request_logs
		WHERE timestamp > now() - $1::interval
	`, interval).Scan(
		&stats.TotalRequests,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCacheReadTokens,
		&stats.TotalCost,
		&stats.AvgLatencyMS,
		&stats.AvgOverheadUS,
		&stats.ErrorCount,
	)
	if err != nil {
		return nil, fmt.Errorf("get overview stats: %w", err)
	}

	if stats.TotalRequests > 0 {
		stats.ErrorRate = float64(stats.ErrorCount) / float64(stats.TotalRequests)
	}

	totalPromptTokens := stats.TotalInputTokens + stats.TotalCacheReadTokens
	if totalPromptTokens > 0 {
		stats.CacheHitRate = float64(stats.TotalCacheReadTokens) / float64(totalPromptTokens)
	}

	return &stats, nil
}

func (s *Store) GetStatsByKey(ctx context.Context, period string, page, perPage int) ([]KeyStats, int, error) {
	interval := periodToInterval(period)
	offset := (page - 1) * perPage

	rows, err := s.pool.Query(ctx, `
		SELECT rl.llm_key_id, k.key_prefix, k.name,
			COUNT(*), COALESCE(SUM(rl.input_tokens), 0), COALESCE(SUM(rl.output_tokens), 0),
			COALESCE(SUM(rl.cost), 0), COALESCE(AVG(rl.latency_ms)::int, 0),
			COUNT(*) OVER() as total
		FROM request_logs rl
		JOIN llm_api_keys k ON k.id = rl.llm_key_id
		WHERE rl.timestamp > now() - $1::interval
		GROUP BY rl.llm_key_id, k.key_prefix, k.name
		ORDER BY SUM(rl.cost) DESC
		LIMIT $2 OFFSET $3
	`, interval, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get stats by key: %w", err)
	}
	defer rows.Close()

	var stats []KeyStats
	var total int
	for rows.Next() {
		var ks KeyStats
		if err := rows.Scan(
			&ks.KeyID, &ks.KeyPrefix, &ks.KeyName,
			&ks.TotalRequests, &ks.TotalInputTokens, &ks.TotalOutputTokens,
			&ks.TotalCost, &ks.AvgLatencyMS,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan key stats: %w", err)
		}
		stats = append(stats, ks)
	}
	return stats, total, rows.Err()
}

func (s *Store) GetStatsByModel(ctx context.Context, period string) ([]ModelStats, error) {
	interval := periodToInterval(period)

	rows, err := s.pool.Query(ctx, `
		SELECT model, COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(cost), 0), COALESCE(AVG(latency_ms)::int, 0)
		FROM request_logs
		WHERE timestamp > now() - $1::interval AND model IS NOT NULL
		GROUP BY model
		ORDER BY SUM(cost) DESC
	`, interval)
	if err != nil {
		return nil, fmt.Errorf("get stats by model: %w", err)
	}
	defer rows.Close()

	var stats []ModelStats
	for rows.Next() {
		var ms ModelStats
		if err := rows.Scan(
			&ms.Model, &ms.TotalRequests, &ms.TotalInputTokens, &ms.TotalOutputTokens,
			&ms.TotalCost, &ms.AvgLatencyMS,
		); err != nil {
			return nil, fmt.Errorf("scan model stats: %w", err)
		}
		stats = append(stats, ms)
	}
	return stats, rows.Err()
}

func (s *Store) GetTimeSeries(ctx context.Context, period, interval string) ([]TimeSeriesBucket, error) {
	pgInterval := periodToInterval(period)
	trunc := intervalToTrunc(interval)

	rows, err := s.pool.Query(ctx, `
		SELECT date_trunc($1, timestamp) as bucket,
			COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(cost), 0), COALESCE(AVG(latency_ms)::int, 0),
			COALESCE(AVG(overhead_us)::int, 0),
			COUNT(*) FILTER (WHERE status_code >= 400)
		FROM request_logs
		WHERE timestamp > now() - $2::interval
		GROUP BY bucket ORDER BY bucket
	`, trunc, pgInterval)
	if err != nil {
		return nil, fmt.Errorf("get time series: %w", err)
	}
	defer rows.Close()

	var buckets []TimeSeriesBucket
	for rows.Next() {
		var b TimeSeriesBucket
		if err := rows.Scan(
			&b.Bucket, &b.Requests, &b.InputTokens, &b.OutputTokens,
			&b.Cost, &b.AvgLatencyMS, &b.AvgOverheadUS, &b.Errors,
		); err != nil {
			return nil, fmt.Errorf("scan time series bucket: %w", err)
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func (s *Store) GetLatencyPercentiles(ctx context.Context, period string) (*LatencyStats, error) {
	interval := periodToInterval(period)
	var stats LatencyStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(percentile_cont(0.50) WITHIN GROUP (ORDER BY latency_ms)::int, 0),
			COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms)::int, 0),
			COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY latency_ms)::int, 0),
			COALESCE(percentile_cont(0.50) WITHIN GROUP (ORDER BY overhead_us)::int, 0),
			COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY overhead_us)::int, 0),
			COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY overhead_us)::int, 0)
		FROM request_logs
		WHERE timestamp > now() - $1::interval AND latency_ms IS NOT NULL
	`, interval).Scan(&stats.P50, &stats.P95, &stats.P99, &stats.OverheadP50US, &stats.OverheadP95US, &stats.OverheadP99US)
	if err != nil {
		return nil, fmt.Errorf("get latency percentiles: %w", err)
	}
	return &stats, nil
}
