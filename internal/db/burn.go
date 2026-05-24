package db

import (
	"context"
	"time"
)

type BurnEvent struct {
	ID          string
	AssetSymbol string
	Amount      int64
	Operator    string
	CreatedAt   time.Time
}

func (d *DB) InsertBurnEvent(ctx context.Context, e BurnEvent) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO burn_events (asset_symbol, amount, operator)
		VALUES ($1, $2, $3)`,
		e.AssetSymbol, e.Amount, e.Operator,
	)
	return err
}

func (d *DB) ListBurnEvents(ctx context.Context, symbol string, limit, offset int) ([]BurnEvent, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, asset_symbol, amount, operator, created_at
		FROM burn_events WHERE asset_symbol = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		symbol, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BurnEvent
	for rows.Next() {
		var e BurnEvent
		if err := rows.Scan(&e.ID, &e.AssetSymbol, &e.Amount, &e.Operator, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
