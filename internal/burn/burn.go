package burn

import (
	"context"
	"fmt"

	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/domain"
	"github.com/houseofhufflepuff/bolt/internal/tapd"
)

type Service struct {
	db   *db.DB
	tapd *tapd.Client
}

func New(database *db.DB, tapdClient *tapd.Client) *Service {
	return &Service{db: database, tapd: tapdClient}
}

func (s *Service) Burn(ctx context.Context, req domain.BurnRequest) error {
	if s.tapd == nil {
		return fmt.Errorf("tapd client not configured")
	}
	asset, err := s.db.GetAsset(ctx, req.AssetSymbol)
	if err != nil {
		return fmt.Errorf("get asset: %w", err)
	}
	if asset.TotalSupply < req.Amount {
		return fmt.Errorf("burn amount %d exceeds supply %d", req.Amount, asset.TotalSupply)
	}
	if err := s.tapd.BurnAsset(ctx, asset.AssetID, req.Amount); err != nil {
		return fmt.Errorf("tapd burn: %w", err)
	}
	if err := s.db.InsertBurnEvent(ctx, db.BurnEvent{
		AssetSymbol: req.AssetSymbol,
		Amount:      req.Amount,
		Operator:    req.Operator,
	}); err != nil {
		return fmt.Errorf("insert burn event: %w", err)
	}
	if err := s.db.AddSupply(ctx, req.AssetSymbol, -req.Amount); err != nil {
		return fmt.Errorf("update supply: %w", err)
	}
	return nil
}
