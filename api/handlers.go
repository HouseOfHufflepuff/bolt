package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/houseofhufflepuff/bolt/internal/burn"
	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/domain"
	"github.com/houseofhufflepuff/bolt/internal/mint"
	"github.com/houseofhufflepuff/bolt/internal/nav"
	"github.com/houseofhufflepuff/bolt/internal/oracle"
	"github.com/houseofhufflepuff/bolt/internal/rebalance"
	"github.com/houseofhufflepuff/bolt/internal/redeem"
	"github.com/houseofhufflepuff/bolt/internal/reserves"
	"github.com/houseofhufflepuff/bolt/internal/webhook"
)

type API struct {
	db           *db.DB
	mint         *mint.Service
	burn         *burn.Service
	redeem       *redeem.Service
	rebalance    *rebalance.Service
	proof        *reserves.Service
	oracle       *oracle.Oracle
	webhook      *webhook.Dispatcher
	genesisUnits map[string]float64
	apiKey       string
}

func New(
	database *db.DB,
	mintSvc *mint.Service,
	burnSvc *burn.Service,
	redeemSvc *redeem.Service,
	rebalanceSvc *rebalance.Service,
	proofSvc *reserves.Service,
	orc *oracle.Oracle,
	webhookDisp *webhook.Dispatcher,
	genesisUnits map[string]float64,
	apiKey string,
) *API {
	return &API{
		db:           database,
		mint:         mintSvc,
		burn:         burnSvc,
		redeem:       redeemSvc,
		rebalance:    rebalanceSvc,
		proof:        proofSvc,
		oracle:       orc,
		webhook:      webhookDisp,
		genesisUnits: genesisUnits,
		apiKey:       apiKey,
	}
}

func (a *API) Routes(r chi.Router) {
	r.Use(LatencyLogger)

	// Public
	r.Get("/supply", a.handleSupply)
	r.Get("/weights", a.handleWeights)
	r.Get("/price", a.handlePrice)
	r.Get("/transactions", a.handleTransactions)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(APIKeyMiddleware(a.apiKey))
		r.Post("/mint", a.handleMint)
		r.Post("/mint/bolt", a.handleMintBOLT)
		r.Post("/burn", a.handleBurn)
		r.Post("/redeem/{asset}", a.handleRedeem)
		r.Get("/reserves", a.handleReserves)
		r.Post("/rebalance", a.handleRebalance)
	})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// --- handlers ---

func (a *API) handleSupply(w http.ResponseWriter, r *http.Request) {
	assets, err := a.db.ListAssets(r.Context())
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	type row struct {
		Symbol      string `json:"symbol"`
		TotalSupply int64  `json:"total_supply"`
	}
	out := make([]row, len(assets))
	for i, a := range assets {
		out[i] = row{Symbol: a.Symbol, TotalSupply: a.TotalSupply}
	}
	writeJSON(w, 200, map[string]any{"assets": out})
}

func (a *API) handleWeights(w http.ResponseWriter, r *http.Request) {
	prices := a.oracle.Prices()
	current := nav.Compute(a.genesisUnits, domain.DefaultWeights, prices)
	writeJSON(w, 200, map[string]any{
		"target_weights":  domain.DefaultWeights.AsMap(),
		"effective_drift": current.WeightDrift,
	})
}

func (a *API) handlePrice(w http.ResponseWriter, r *http.Request) {
	prices := a.oracle.Prices()
	current := nav.Compute(a.genesisUnits, domain.DefaultWeights, prices)
	writeJSON(w, 200, map[string]any{
		"nav_usd":    current.TotalUSD,
		"breakdown":  current.Breakdown,
		"prices":     prices,
	})
}

func (a *API) handleMint(w http.ResponseWriter, r *http.Request) {
	var req domain.MintRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	result, err := a.mint.Mint(r.Context(), req)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	a.webhook.Send(r.Context(), webhook.Event{Type: "mint", Payload: req})
	writeJSON(w, 200, map[string]any{
		"batch_key":   hex.EncodeToString(result.BatchKey),
		"asset_proof": hex.EncodeToString(result.AssetProof),
	})
}

func (a *API) handleMintBOLT(w http.ResponseWriter, r *http.Request) {
	var req mint.BOLTMintRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	prices := a.oracle.Prices()
	result, err := a.mint.MintBOLT(r.Context(), req, prices, a.genesisUnits)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	a.webhook.Send(r.Context(), webhook.Event{Type: "mint_bolt", Payload: req})
	writeJSON(w, 200, map[string]any{
		"batch_key":   hex.EncodeToString(result.BatchKey),
		"asset_proof": hex.EncodeToString(result.AssetProof),
	})
}

func (a *API) handleBurn(w http.ResponseWriter, r *http.Request) {
	var req domain.BurnRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if err := a.burn.Burn(r.Context(), req); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	a.webhook.Send(r.Context(), webhook.Event{Type: "burn", Payload: req})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *API) handleRedeem(w http.ResponseWriter, r *http.Request) {
	asset := chi.URLParam(r, "asset")
	var req redeem.Request
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	req.AssetSymbol = asset
	if err := a.redeem.Redeem(r.Context(), req); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	a.webhook.Send(r.Context(), webhook.Event{Type: "redeem", Payload: req})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (a *API) handleReserves(w http.ResponseWriter, r *http.Request) {
	assets, err := a.db.ListAssets(r.Context())
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	proof, err := a.proof.Snapshot(r.Context(), assets)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	type snapRow struct {
		Asset   string  `json:"asset"`
		Reserve int64   `json:"reserve"`
		Supply  int64   `json:"supply"`
		Ratio   float64 `json:"ratio"`
	}
	rows := make([]snapRow, len(proof.Snapshots))
	for i, s := range proof.Snapshots {
		rows[i] = snapRow{Asset: s.AssetSymbol, Reserve: s.ReserveAmount, Supply: s.SupplyAmount, Ratio: s.Ratio}
	}
	writeJSON(w, 200, map[string]any{
		"snapshots":   rows,
		"merkle_root": hex.EncodeToString(proof.MerkleRoot),
		"signature":   hex.EncodeToString(proof.Signature),
	})
}

func (a *API) handleRebalance(w http.ResponseWriter, r *http.Request) {
	prices := a.oracle.Prices()
	result, err := a.rebalance.Rebalance(r.Context(), prices, a.genesisUnits, "api")
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	a.webhook.Send(r.Context(), webhook.Event{Type: "rebalance", Payload: result})
	writeJSON(w, 200, map[string]any{
		"rebalance_id": result.ID,
		"trade_queue":  result.Trades,
		"nav_usd":      result.CurrentNAV.TotalUSD,
		"drift":        result.CurrentNAV.WeightDrift,
	})
}

func (a *API) handleTransactions(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit == 0 {
		limit = 20
	}
	mints, err := a.db.ListMintEvents(r.Context(), symbol, limit, offset)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	burns, err := a.db.ListBurnEvents(r.Context(), symbol, limit, offset)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{
		"mint_events": mints,
		"burn_events": burns,
	})
}
