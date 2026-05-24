package rebalance

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/domain"
	"github.com/houseofhufflepuff/bolt/internal/nav"
)

type Trade struct {
	Symbol   string  `json:"symbol"`
	Action   string  `json:"action"`   // "buy" | "sell"
	DriftPct float64 `json:"drift_pct"` // positive = over-weight, negative = under-weight
}

type Result struct {
	ID         string
	Trades     []Trade
	CurrentNAV domain.BasketNAV
}

type Service struct {
	db *db.DB
}

func New(database *db.DB) *Service {
	return &Service{db: database}
}

// Rebalance computes weight drift, records a trade queue, and returns it for execution.
// Assets with drift < 1% are skipped. Execution of trades is out of scope for the MVP.
func (s *Service) Rebalance(ctx context.Context, prices map[string]float64, genesisUnits map[string]float64, triggeredBy string) (*Result, error) {
	current := nav.Compute(genesisUnits, domain.DefaultWeights, prices)

	var trades []Trade
	for symbol, drift := range current.WeightDrift {
		if math.Abs(drift) < 0.01 {
			continue
		}
		action := "buy"
		if drift > 0 {
			action = "sell"
		}
		trades = append(trades, Trade{
			Symbol:   symbol,
			Action:   action,
			DriftPct: drift * 100,
		})
	}

	beforeJSON, _ := json.Marshal(current.WeightDrift)
	afterJSON, _ := json.Marshal(domain.DefaultWeights.AsMap())
	tradesJSON, _ := json.Marshal(trades)

	id, err := s.db.InsertRebalanceLog(ctx, db.RebalanceLog{
		TriggeredBy:   triggeredBy,
		WeightsBefore: beforeJSON,
		WeightsAfter:  afterJSON,
		TradeQueue:    tradesJSON,
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("insert rebalance log: %w", err)
	}

	return &Result{ID: id, Trades: trades, CurrentNAV: current}, nil
}
