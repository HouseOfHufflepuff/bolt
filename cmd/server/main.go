package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/houseofhufflepuff/bolt/api"
	"github.com/houseofhufflepuff/bolt/internal/burn"
	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/domain"
	"github.com/houseofhufflepuff/bolt/internal/lnd"
	"github.com/houseofhufflepuff/bolt/internal/mint"
	"github.com/houseofhufflepuff/bolt/internal/nav"
	"github.com/houseofhufflepuff/bolt/internal/oracle"
	"github.com/houseofhufflepuff/bolt/internal/rebalance"
	"github.com/houseofhufflepuff/bolt/internal/redeem"
	"github.com/houseofhufflepuff/bolt/internal/reserve"
	"github.com/houseofhufflepuff/bolt/internal/reserves"
	"github.com/houseofhufflepuff/bolt/internal/tapd"
	"github.com/houseofhufflepuff/bolt/internal/webhook"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	ctx := context.Background()

	// --- DB ---
	database, err := db.New(ctx, env("BOLT_DB_URL", "postgresql://root@localhost:26257/bolt?sslmode=disable"))
	if err != nil {
		slog.Warn("db unavailable (supply/transactions endpoints will fail)", "err", err)
	}

	// --- LND (optional: needed for BTC reserve + mint/burn/redeem) ---
	var lndClient *lnd.Client
	if host := env("BOLT_LND_HOST", ""); host != "" {
		lndClient, err = lnd.New(host,
			env("BOLT_LND_CERT", ""),
			env("BOLT_LND_MACAROON", ""),
		)
		if err != nil {
			slog.Warn("lnd connect failed", "err", err)
		}
	}

	// --- tapd (optional: needed for mint/burn) ---
	var tapdClient *tapd.Client
	if host := env("BOLT_TAPD_HOST", ""); host != "" {
		tapdClient, err = tapd.New(host,
			env("BOLT_TAPD_CERT", ""),
			env("BOLT_TAPD_MACAROON", ""),
		)
		if err != nil {
			slog.Warn("tapd connect failed", "err", err)
		}
	}

	// --- Oracle (always runs; needs internet for Coinbase prices) ---
	orc := oracle.New(database, 30*time.Second)
	go orc.Run(ctx)

	// Wait for first oracle fetch, then compute genesis units.
	time.Sleep(2 * time.Second)
	prices := orc.Prices()
	genesisUnits := nav.GenesisUnits(domain.DefaultWeights, prices)
	if len(genesisUnits) == 0 {
		slog.Warn("oracle prices unavailable, using dev genesis units")
		genesisUnits = devGenesisUnits()
	}

	// --- Services ---
	reserveChecker := reserve.New(lndClient)
	mintSvc := mint.New(database, tapdClient, reserveChecker)
	burnSvc := burn.New(database, tapdClient)
	redeemSvc := redeem.New(burnSvc, lndClient)
	rebalanceSvc := rebalance.New(database)
	proofSvc := reserves.New(database, reserveChecker, nil) // nil = generate ephemeral key; load from file in prod
	webhookDisp := webhook.New(env("BOLT_WEBHOOK_SECRET", "dev-secret"))

	// --- API ---
	handler := api.New(
		database, mintSvc, burnSvc, redeemSvc, rebalanceSvc, proofSvc,
		orc, webhookDisp, genesisUnits,
		env("BOLT_API_KEY", "dev-key"),
	)

	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})
	handler.Routes(r)

	addr := env("BOLT_ADDR", ":8080")
	slog.Info("bolt starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

// devGenesisUnits are approximate basket unit counts used when oracle is unavailable.
// Computed as target_weight / reference_price:
//   USDL: 0.50 / 1.00 = 0.500000
//   CHFL: 0.30 / 1.09 = 0.275229
//   NOKL: 0.10 / 0.093 = 1.075269
//   BTC:  0.10 / 65000 = 0.0000015
func devGenesisUnits() map[string]float64 {
	return map[string]float64{
		domain.SymbolUSDL: 0.500000,
		domain.SymbolCHFL: 0.275229,
		domain.SymbolNOKL: 1.075269,
		domain.SymbolBTC:  0.0000015,
	}
}
