-- Backfill costs for request logs based on current model pricing
UPDATE request_logs rl
SET cost = (
    COALESCE(rl.input_tokens, 0) * m.input_cost_per_million / 1000000.0 +
    COALESCE(rl.output_tokens, 0) * m.output_cost_per_million / 1000000.0
)
FROM models m
WHERE rl.model = m.name
  AND rl.cost = 0
  AND (rl.input_tokens IS NOT NULL OR rl.output_tokens IS NOT NULL);

-- Show summary
SELECT
    COUNT(*) as updated_logs,
    SUM(cost) as total_backfilled_cost
FROM request_logs
WHERE cost > 0;
