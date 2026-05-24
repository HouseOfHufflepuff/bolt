package db

import (
	"context"
	"time"
)

type MintEvent struct {
	ID          string
	AssetSymbol string
	Amount      int64
	Operator    string
	BatchKey    []byte
	AssetProof  []byte
	CreatedAt   time.Time
}

func (d *DB) InsertMintEvent(ctx context.Context, e MintEvent) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO mint_events (asset_symbol, amount, operator, batch_key, asset_proof)
		VALUES ($1, $2, $3, $4, $5)`,
		e.AssetSymbol, e.Amount, e.Operator, e.BatchKey, e.AssetProof,
	)
	return err
}

func (d *DB) ListMintEvents(ctx context.Context, symbol string, limit, offset int) ([]MintEvent, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, asset_symbol, amount, operator, batch_key, asset_proof, created_at
		FROM mint_events WHERE asset_symbol = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		symbol, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MintEvent
	for rows.Next() {
		var e MintEvent
		if err := rows.Scan(&e.ID, &e.AssetSymbol, &e.Amount, &e.Operator, &e.BatchKey, &e.AssetProof, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
