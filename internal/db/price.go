package db

import (
	"context"
	"time"
)

type PriceEntry struct {
	ID          string
	AssetSymbol string
	PriceUSD    float64
	Source      string
	RecordedAt  time.Time
}

func (d *DB) InsertPrice(ctx context.Context, e PriceEntry) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO price_feed (asset_symbol, price_usd, source, recorded_at)
		VALUES ($1, $2, $3, $4)`,
		e.AssetSymbol, e.PriceUSD, e.Source, e.RecordedAt,
	)
	return err
}

func (d *DB) GetLatestPrice(ctx context.Context, symbol string) (PriceEntry, error) {
	var e PriceEntry
	err := d.pool.QueryRow(ctx, `
		SELECT id, asset_symbol, price_usd, source, recorded_at
		FROM price_feed WHERE asset_symbol = $1
		ORDER BY recorded_at DESC LIMIT 1`, symbol,
	).Scan(&e.ID, &e.AssetSymbol, &e.PriceUSD, &e.Source, &e.RecordedAt)
	return e, err
}

func (d *DB) GetLatestPrices(ctx context.Context, symbols []string) (map[string]PriceEntry, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT DISTINCT ON (asset_symbol) id, asset_symbol, price_usd, source, recorded_at
		FROM price_feed WHERE asset_symbol = ANY($1)
		ORDER BY asset_symbol, recorded_at DESC`, symbols,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]PriceEntry, len(symbols))
	for rows.Next() {
		var e PriceEntry
		if err := rows.Scan(&e.ID, &e.AssetSymbol, &e.PriceUSD, &e.Source, &e.RecordedAt); err != nil {
			return nil, err
		}
		out[e.AssetSymbol] = e
	}
	return out, rows.Err()
}
