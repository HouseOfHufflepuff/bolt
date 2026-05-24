package db

import (
	"context"
	"time"
)

type RebalanceLog struct {
	ID             string
	TriggeredBy    string
	WeightsBefore  []byte
	WeightsAfter   []byte
	TradeQueue     []byte
	Status         string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

func (d *DB) InsertRebalanceLog(ctx context.Context, r RebalanceLog) (string, error) {
	var id string
	err := d.pool.QueryRow(ctx, `
		INSERT INTO rebalance_log (triggered_by, weights_before, weights_after, trade_queue, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		r.TriggeredBy, r.WeightsBefore, r.WeightsAfter, r.TradeQueue, r.Status,
	).Scan(&id)
	return id, err
}

func (d *DB) UpdateRebalanceStatus(ctx context.Context, id, status string, completedAt *time.Time) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE rebalance_log SET status = $1, completed_at = $2 WHERE id = $3`,
		status, completedAt, id,
	)
	return err
}

func (d *DB) ListRebalanceLogs(ctx context.Context, limit, offset int) ([]RebalanceLog, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, triggered_by, weights_before, weights_after, trade_queue, status, created_at, completed_at
		FROM rebalance_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RebalanceLog
	for rows.Next() {
		var r RebalanceLog
		if err := rows.Scan(&r.ID, &r.TriggeredBy, &r.WeightsBefore, &r.WeightsAfter, &r.TradeQueue, &r.Status, &r.CreatedAt, &r.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
