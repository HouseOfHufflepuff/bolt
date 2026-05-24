package mint

import (
	"context"
	"fmt"

	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/domain"
	"github.com/houseofhufflepuff/bolt/internal/nav"
)

type BOLTMintRequest struct {
	USDL     int64  // base units of USDL to contribute
	CHFL     int64  // base units of CHFL to contribute
	NOKL     int64  // base units of NOKL to contribute
	BTC      int64  // satoshis
	Operator string
}

// MintBOLT issues BOLT tokens proportional to the USD value of the provided constituents
// at the current basket NAV. genesisUnits is the per-asset unit count fixed at platform launch.
func (s *Service) MintBOLT(ctx context.Context, req BOLTMintRequest, prices map[string]float64, genesisUnits map[string]float64) (domain.MintResult, error) {
	current := nav.Compute(genesisUnits, domain.DefaultWeights, prices)
	if current.TotalUSD <= 0 {
		return domain.MintResult{}, fmt.Errorf("NAV unavailable (%.6f) — oracle may not have prices yet", current.TotalUSD)
	}

	contributed := float64(req.USDL)*prices[domain.SymbolUSDL] +
		float64(req.CHFL)*prices[domain.SymbolCHFL] +
		float64(req.NOKL)*prices[domain.SymbolNOKL] +
		float64(req.BTC)*prices[domain.SymbolBTC]

	// BOLT amount in base units (8 decimal places)
	boltAmount := int64(contributed / current.TotalUSD * 1e8)
	if boltAmount <= 0 {
		return domain.MintResult{}, fmt.Errorf("contributed value too small to issue BOLT (contributed $%.6f, NAV $%.6f)", contributed, current.TotalUSD)
	}

	// BOLT reserve check is implicit: the constituents backing it must already be in reserve.
	// For the MVP we trust the operator has locked the constituent tokens.
	if s.tapd == nil {
		return domain.MintResult{}, fmt.Errorf("tapd client not configured")
	}

	batch, err := s.tapd.MintAsset(ctx, domain.SymbolBOLT, boltAmount)
	if err != nil {
		return domain.MintResult{}, fmt.Errorf("tapd mint BOLT: %w", err)
	}
	proof, err := s.tapd.FinalizeBatch(ctx)
	if err != nil {
		return domain.MintResult{}, fmt.Errorf("tapd finalize: %w", err)
	}

	if err := s.db.InsertMintEvent(ctx, db.MintEvent{
		AssetSymbol: domain.SymbolBOLT,
		Amount:      boltAmount,
		Operator:    req.Operator,
		BatchKey:    batch.BatchKey,
		AssetProof:  proof,
	}); err != nil {
		return domain.MintResult{}, fmt.Errorf("insert mint event: %w", err)
	}
	if err := s.db.AddSupply(ctx, domain.SymbolBOLT, boltAmount); err != nil {
		return domain.MintResult{}, fmt.Errorf("update supply: %w", err)
	}

	return domain.MintResult{BatchKey: batch.BatchKey, AssetProof: proof}, nil
}
