package db

import (
	"context"
	"time"
)

type Asset struct {
	ID           string
	Symbol       string
	Name         string
	Decimals     int
	AssetID      []byte
	GenesisPoint string
	TotalSupply  int64
	CreatedAt    time.Time
}

func (d *DB) UpsertAsset(ctx context.Context, a Asset) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO assets (symbol, name, decimals, asset_id, genesis_point, total_supply)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (symbol) DO UPDATE SET
			asset_id      = EXCLUDED.asset_id,
			genesis_point = EXCLUDED.genesis_point`,
		a.Symbol, a.Name, a.Decimals, a.AssetID, a.GenesisPoint, a.TotalSupply,
	)
	return err
}

func (d *DB) GetAsset(ctx context.Context, symbol string) (Asset, error) {
	var a Asset
	err := d.pool.QueryRow(ctx, `
		SELECT id, symbol, name, decimals, asset_id, genesis_point, total_supply, created_at
		FROM assets WHERE symbol = $1`, symbol,
	).Scan(&a.ID, &a.Symbol, &a.Name, &a.Decimals, &a.AssetID, &a.GenesisPoint, &a.TotalSupply, &a.CreatedAt)
	return a, err
}

func (d *DB) ListAssets(ctx context.Context) ([]Asset, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, symbol, name, decimals, asset_id, genesis_point, total_supply, created_at
		FROM assets ORDER BY symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.Symbol, &a.Name, &a.Decimals, &a.AssetID, &a.GenesisPoint, &a.TotalSupply, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (d *DB) AddSupply(ctx context.Context, symbol string, delta int64) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE assets SET total_supply = total_supply + $1 WHERE symbol = $2`,
		delta, symbol,
	)
	return err
}
