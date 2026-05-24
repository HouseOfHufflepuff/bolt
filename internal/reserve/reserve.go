package reserve

import (
	"context"
	"fmt"
	"sync"

	"github.com/houseofhufflepuff/bolt/internal/lnd"
)

// Checker verifies that reserves cover supply before minting.
type Checker struct {
	lnd  *lnd.Client
	mu   sync.RWMutex
	fiat map[string]int64 // stub until real banking API is wired
}

func New(lndClient *lnd.Client) *Checker {
	return &Checker{
		lnd:  lndClient,
		fiat: make(map[string]int64),
	}
}

// SetFiatBalance sets the reserve balance for a fiat-pegged asset.
// Replace with a real banking API connector per asset.
func (c *Checker) SetFiatBalance(symbol string, amount int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fiat[symbol] = amount
}

// Check returns an error if minting amount would push supply above reserves.
func (c *Checker) Check(ctx context.Context, symbol string, currentSupply, mintAmount int64) error {
	reserve, err := c.Balance(ctx, symbol)
	if err != nil {
		return err
	}
	projected := currentSupply + mintAmount
	if projected == 0 {
		return nil
	}
	ratio := float64(reserve) / float64(projected)
	if ratio < 1.0 {
		return fmt.Errorf("%s reserve ratio %.4f < 1.0 (reserve=%d projected_supply=%d)",
			symbol, ratio, reserve, projected)
	}
	return nil
}

// Balance returns the confirmed reserve balance for the asset in base units.
func (c *Checker) Balance(ctx context.Context, symbol string) (int64, error) {
	if symbol == "BTC" {
		bal, err := c.lnd.WalletBalance(ctx)
		if err != nil {
			return 0, fmt.Errorf("lnd wallet balance: %w", err)
		}
		return bal.Confirmed, nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	bal, ok := c.fiat[symbol]
	if !ok {
		return 0, fmt.Errorf("no reserve configured for %s", symbol)
	}
	return bal, nil
}
