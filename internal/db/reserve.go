package db

import (
	"context"
	"time"
)

type ReserveSnapshot struct {
	ID            string
	AssetSymbol   string
	ReserveAmount int64
	SupplyAmount  int64
	Ratio         float64
	MerkleRoot    []byte
	Signature     []byte
	CreatedAt     time.Time
}

func (d *DB) InsertReserveSnapshot(ctx context.Context, s ReserveSnapshot) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO reserve_snapshots (asset_symbol, reserve_amount, supply_amount, ratio, merkle_root, signature)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		s.AssetSymbol, s.ReserveAmount, s.SupplyAmount, s.Ratio, s.MerkleRoot, s.Signature,
	)
	return err
}

func (d *DB) GetLatestSnapshot(ctx context.Context, symbol string) (ReserveSnapshot, error) {
	var s ReserveSnapshot
	err := d.pool.QueryRow(ctx, `
		SELECT id, asset_symbol, reserve_amount, supply_amount, ratio, merkle_root, signature, created_at
		FROM reserve_snapshots WHERE asset_symbol = $1
		ORDER BY created_at DESC LIMIT 1`, symbol,
	).Scan(&s.ID, &s.AssetSymbol, &s.ReserveAmount, &s.SupplyAmount, &s.Ratio, &s.MerkleRoot, &s.Signature, &s.CreatedAt)
	return s, err
}
