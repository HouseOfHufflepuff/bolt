package redeem

import (
	"context"
	"fmt"

	"github.com/houseofhufflepuff/bolt/internal/burn"
	"github.com/houseofhufflepuff/bolt/internal/domain"
	"github.com/houseofhufflepuff/bolt/internal/lnd"
)

type Service struct {
	burn *burn.Service
	lnd  *lnd.Client
}

func New(burnSvc *burn.Service, lndClient *lnd.Client) *Service {
	return &Service{burn: burnSvc, lnd: lndClient}
}

type Request struct {
	AssetSymbol string
	Amount      int64
	Operator    string
	// BTC: Lightning payment request string. Fiat: queued for wire (stub).
	Destination string
}

// Redeem burns the asset tokens and dispatches the underlying value to the destination.
// Fiat redemptions (USDL, CHFL, NOKL) are stubbed — wire transfer integration needed.
func (s *Service) Redeem(ctx context.Context, req Request) error {
	if err := s.burn.Burn(ctx, domain.BurnRequest{
		AssetSymbol: req.AssetSymbol,
		Amount:      req.Amount,
		Operator:    req.Operator,
	}); err != nil {
		return fmt.Errorf("burn on redeem: %w", err)
	}

	switch req.AssetSymbol {
	case domain.SymbolBTC:
		if s.lnd == nil {
			return fmt.Errorf("lnd client not configured for BTC redemption")
		}
		if _, err := s.lnd.SendPaymentSync(ctx, req.Destination); err != nil {
			return fmt.Errorf("lnd send: %w", err)
		}
	case domain.SymbolUSDL, domain.SymbolCHFL, domain.SymbolNOKL:
		// Stub: queue wire transfer via banking API. Tokens are already burned above.
		// TODO: integrate banking partner API per asset currency.
	default:
		return fmt.Errorf("unsupported asset for redemption: %s", req.AssetSymbol)
	}
	return nil
}
