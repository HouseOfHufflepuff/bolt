package mint

import (
	"context"
	"fmt"

	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/domain"
	"github.com/houseofhufflepuff/bolt/internal/reserve"
	"github.com/houseofhufflepuff/bolt/internal/tapd"
)

type Service struct {
	db      *db.DB
	tapd    *tapd.Client
	reserve *reserve.Checker
}

func New(database *db.DB, tapdClient *tapd.Client, checker *reserve.Checker) *Service {
	return &Service{db: database, tapd: tapdClient, reserve: checker}
}

// Mint validates reserves, calls tapd to mint the asset, records the event, and updates supply.
func (s *Service) Mint(ctx context.Context, req domain.MintRequest) (domain.MintResult, error) {
	asset, err := s.db.GetAsset(ctx, req.AssetSymbol)
	if err != nil {
		return domain.MintResult{}, fmt.Errorf("get asset %s: %w", req.AssetSymbol, err)
	}

	if err := s.reserve.Check(ctx, req.AssetSymbol, asset.TotalSupply, req.Amount); err != nil {
		return domain.MintResult{}, fmt.Errorf("reserve check: %w", err)
	}

	batch, err := s.tapd.MintAsset(ctx, req.AssetSymbol, req.Amount)
	if err != nil {
		return domain.MintResult{}, fmt.Errorf("tapd mint: %w", err)
	}

	proof, err := s.tapd.FinalizeBatch(ctx)
	if err != nil {
		return domain.MintResult{}, fmt.Errorf("tapd finalize: %w", err)
	}

	if err := s.db.InsertMintEvent(ctx, db.MintEvent{
		AssetSymbol: req.AssetSymbol,
		Amount:      req.Amount,
		Operator:    req.Operator,
		BatchKey:    batch.BatchKey,
		AssetProof:  proof,
	}); err != nil {
		return domain.MintResult{}, fmt.Errorf("insert mint event: %w", err)
	}

	if err := s.db.AddSupply(ctx, req.AssetSymbol, req.Amount); err != nil {
		return domain.MintResult{}, fmt.Errorf("update supply: %w", err)
	}

	return domain.MintResult{BatchKey: batch.BatchKey, AssetProof: proof}, nil
}
