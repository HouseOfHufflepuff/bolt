package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/domain"
)

const coinbaseSpot = "https://api.coinbase.com/v2/prices/%s/spot"

// pairs maps our asset symbol to the Coinbase price pair.
// USD is always 1.0 — no fetch needed.
var pairs = map[string]string{
	domain.SymbolBTC:  "BTC-USD",
	domain.SymbolCHFL: "CHF-USD",
	domain.SymbolNOKL: "NOK-USD",
}

type Oracle struct {
	db       *db.DB
	mu       sync.RWMutex
	cache    map[string]float64
	interval time.Duration
}

func New(database *db.DB, interval time.Duration) *Oracle {
	return &Oracle{
		db:       database,
		cache:    map[string]float64{domain.SymbolUSDL: 1.0},
		interval: interval,
	}
}

// Run polls Coinbase on interval until ctx is cancelled.
func (o *Oracle) Run(ctx context.Context) {
	o.fetch(ctx)
	t := time.NewTicker(o.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			o.fetch(ctx)
		}
	}
}

// GetPrice returns the latest cached USD price for the given asset symbol.
func (o *Oracle) GetPrice(symbol string) (float64, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	p, ok := o.cache[symbol]
	return p, ok
}

// Prices returns a copy of the full price cache.
func (o *Oracle) Prices() map[string]float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make(map[string]float64, len(o.cache))
	for k, v := range o.cache {
		out[k] = v
	}
	return out
}

func (o *Oracle) fetch(ctx context.Context) {
	for symbol, pair := range pairs {
		price, err := fetchSpot(ctx, pair)
		if err != nil {
			continue
		}
		o.mu.Lock()
		o.cache[symbol] = price
		o.mu.Unlock()

		_ = o.db.InsertPrice(ctx, db.PriceEntry{
			AssetSymbol: symbol,
			PriceUSD:    price,
			Source:      "coinbase",
			RecordedAt:  time.Now().UTC(),
		})
	}
}

func fetchSpot(ctx context.Context, pair string) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf(coinbaseSpot, pair), nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var body struct {
		Data struct {
			Amount string `json:"amount"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(body.Data.Amount, 64)
}
