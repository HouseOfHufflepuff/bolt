# Bolt

Asset issuance platform on Bitcoin Lightning using the Taproot Assets Protocol. Issues BOLT — a basket currency backed by USDL, CHFL, NOKL, and BTC — and the constituent assets individually.

---

## BOLT Token

BOLT is a basket-of-assets currency. One BOLT is a redeemable claim on a weighted pool:

| Asset | Weight | Description |
|-------|--------|-------------|
| USDL  | 50%    | USD-pegged stablecoin |
| CHFL  | 30%    | CHF-pegged stablecoin |
| NOKL  | 10%    | NOK-pegged stablecoin |
| BTC   | 10%    | Native Lightning (sats) |

NAV floats with market prices. Weights drift with price movement and rebalance on demand. Each constituent is individually redeemable via API.

**Why CHF:** SNB is constitutionally mandated for price stability, ~1.4% avg inflation vs USD at ~4.5%. Strongest purchasing power preserver among liquid currencies. EUR excluded — expansionary monetary union with no coherent mandate.

**Why NOK:** Norges Bank manages one of the largest sovereign wealth funds globally. NOK is commodity-backed (oil), low debt-to-GDP, disciplined monetary policy.

---

## Stack

| Component | Image | Port |
|-----------|-------|------|
| Bitcoin Core (regtest) | `lncm/bitcoind:v28.0` | 18443 RPC |
| LND (issuer node) | `lightninglabs/lnd:v0.20.1-beta` | 10009 gRPC, 8180 REST |
| LND (receiver node) | `lightninglabs/lnd:v0.20.1-beta` | 10010 gRPC, 8181 REST |
| tapd | `lightninglabs/taproot-assets:v0.7.2` | 10029 gRPC, 8089 REST |
| CockroachDB | `cockroachdb/cockroach:v23.2.5` | 26257 |
| Bolt server | Go 1.22 | 8080 |

LND and tapd clients are REST-based (no proto compilation needed to run).

---

## Prerequisites

```
docker, docker compose
go 1.22
goose       # go install github.com/pressly/goose/v3/cmd/goose@latest
cockroach   # https://www.cockroachlabs.com/docs/stable/install-cockroachdb-mac
jq
```

---

## Running locally

### 1. Start the demo stack

```bash
make demo-start
```

This runs in sequence:
1. `docker compose up -d` — starts all five containers
2. Waits 15s for initialization
3. `scripts/bootstrap.sh` — waits for LND, mines 101 blocks (coinbase maturity), funds LND wallet
4. Creates the `bolt` database and runs goose migrations
5. `scripts/open-channel.sh` — peers the two LND nodes, opens a 5M sat channel, mines 3 confirmation blocks
6. `scripts/init-assets.sh` — mints genesis USDL/CHFL/NOKL/BOLT in tapd, finalizes, seeds asset records in CockroachDB

### 2. Start the server

```bash
BOLT_LND_HOST=localhost:10009 \
BOLT_LND_CERT=~/.lnd/tls.cert \
BOLT_LND_MACAROON=~/.lnd/data/chain/bitcoin/regtest/admin.macaroon \
BOLT_TAPD_HOST=localhost:10029 \
BOLT_TAPD_CERT=~/.tapd/tls.cert \
BOLT_TAPD_MACAROON=~/.tapd/data/chain/bitcoin/regtest/admin.macaroon \
make run
```

Or without LND/tapd (price/supply/weights endpoints still work, mint/burn/redeem will error):

```bash
make run
```

### 3. Verify

```bash
make health     # → ok
make price      # NAV + breakdown + oracle prices
make weights    # target weights + current drift
make supply     # per-asset circulating supply
```

### Reset everything

```bash
make demo-reset   # docker compose down -v (wipes all volumes)
```

---

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BOLT_ADDR` | `:8080` | Server listen address |
| `BOLT_API_KEY` | `dev-key` | X-API-Key for protected endpoints |
| `BOLT_DB_URL` | `postgresql://root@localhost:26257/bolt?sslmode=disable` | CockroachDB connection |
| `BOLT_LND_HOST` | _(empty — LND disabled)_ | LND REST host:port |
| `BOLT_LND_CERT` | | Path to LND TLS cert |
| `BOLT_LND_MACAROON` | | Path to LND admin macaroon |
| `BOLT_TAPD_HOST` | _(empty — tapd disabled)_ | tapd REST host:port |
| `BOLT_TAPD_CERT` | | Path to tapd TLS cert |
| `BOLT_TAPD_MACAROON` | | Path to tapd admin macaroon |
| `BOLT_WEBHOOK_SECRET` | `dev-secret` | HMAC secret for webhook signatures |

---

## API

### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness check |
| GET | `/supply` | Circulating supply per asset |
| GET | `/price` | BOLT NAV + oracle prices |
| GET | `/weights` | Target weights + current drift |
| GET | `/transactions?symbol=USDL&limit=20` | Mint/burn history |

### Protected (X-API-Key header required)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/mint` | Mint a constituent asset |
| POST | `/mint/bolt` | Mint BOLT from constituent deposits |
| POST | `/burn` | Burn tokens |
| POST | `/redeem/{asset}` | Burn + pay out (BTC via Lightning, fiat stubbed) |
| GET | `/reserves` | Snapshot + signed Merkle proof |
| POST | `/rebalance` | Compute drift, log trade queue |

---

## Make commands

```bash
# Stack
make up                                  # docker compose up -d
make down                                # docker compose down
make demo-start                          # full demo: up + fund + channel + assets
make demo-reset                          # down -v (wipe volumes)

# Mining (regtest)
make mine N=5                            # mine 5 blocks

# Queries (server must be running)
make health
make supply
make price
make weights
make transactions SYMBOL=USDL

# Mutations (require X-API-Key)
make mint SYMBOL=USDL AMOUNT=1000000 OPERATOR=admin
make mint-bolt USDL=500000 CHFL=300000 NOKL=100000 BTC=1540
make burn SYMBOL=USDL AMOUNT=500000 OPERATOR=admin
make redeem SYMBOL=BTC AMOUNT=1000 DEST=lnbc1...
make redeem SYMBOL=USDL AMOUNT=500000
make reserves
make rebalance

# Tests
make test-integration   # requires demo-start + server running
```

---

## Webhooks

Register a URL via `dispatcher.Register(url)` in code. Every mutating operation (mint, burn, redeem, rebalance) posts an event:

```json
{
  "type": "mint",
  "payload": { "asset_symbol": "USDL", "amount": 1000000, "operator": "admin" }
}
```

Each request includes `X-Bolt-Signature: sha256=<hmac-sha256>` signed with `BOLT_WEBHOOK_SECRET`.

---

## Reserve proof

`GET /reserves` returns a SHA-256 Merkle tree over all asset snapshots, signed with an ed25519 key. In production, load the key from a file (pass `privKey` to `reserves.New`). Default is an ephemeral key (changes on restart).

```json
{
  "snapshots": [{ "asset": "BTC", "reserve": 154000, "supply": 100000, "ratio": 1.54 }],
  "merkle_root": "abc123...",
  "signature": "def456..."
}
```

---

## NAV calculation

Genesis units are computed at first boot: `units[asset] = target_weight / genesis_price`.

Current NAV: `sum(units[asset] * current_price[asset])`.

Drift per asset: `effective_weight - target_weight`. Rebalance when |drift| > 1%.

Oracle: Coinbase REST API (`/v2/prices/{pair}/spot`), polled every 30s. USDL hardcoded at 1.0.

---

## Database migrations

```bash
make db-init    # CREATE DATABASE IF NOT EXISTS bolt
make migrate    # goose up (requires BOLT_DB_URL env var)
```

Migrations live in `db/migrations/`. Uses Goose with plain SQL.

---

## Proto generation (optional)

The Go clients use REST. Proto stubs are only needed for gRPC:

```bash
make proto-gen   # fetches tapd + LND protos, runs protoc
```

Outputs to `internal/tapd/pb/` and `internal/lnd/pb/`.
