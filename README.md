# Bolt — Bitcoin Lightning Asset Factory

Bolt is an asset issuance and management platform built on the Bitcoin Lightning Network using the [Taproot Assets Protocol (TAP)](https://docs.lightning.engineering/the-lightning-network/taproot-assets). It enables the minting, management, and transfer of real-world assets — beginning with a USD-pegged stablecoin — as first-class assets on Lightning, enabling near-instant, low-fee, globally accessible payments.

> **MVP Goal:** A USD-pegged stablecoin minted and managed on Bitcoin Lightning Network using the Taproot Assets Protocol — mirroring the architecture Tether used for USDt on Lightning.

---

## Language: Go

**Go is the correct language for this project — no real debate.**

The entire Lightning + Taproot stack is written in Go:

| Dependency | Language |
|---|---|
| `lnd` (Lightning Network Daemon) | Go |
| `tapd` (Taproot Assets Daemon) | Go |
| `btcd` (Bitcoin library) | Go |
| gRPC client stubs (LND + tapd) | Go-native codegen |

Choosing Go means you consume these libraries directly, compile against the same gRPC protobuf definitions, and write idiomatic integrations rather than crossing an FFI or HTTP boundary just to talk to your own node. The alternative (Node, Python, Rust) all require either a REST sidecar or painful FFI — Go does not.

Go also gives you: goroutines for concurrent payment handling, strong typing for satoshi/msat arithmetic (no float bugs), and a single statically-linked binary that deploys cleanly inside Docker alongside `tapd` and `lnd`.

**Frontend:** Next.js (React + TypeScript) for the admin dashboard and user wallet. No debate here either — this is a web UI, and the Go API serves it over REST/gRPC-web.

---

## Why Lightning + Taproot Assets?

- **Taproot Assets** allows asset issuance directly on Bitcoin, with transfers routed through existing Lightning channels.
- **No new chain.** Assets settle on Bitcoin. No side-chain trust assumptions.
- **Instant finality.** Lightning payments settle in milliseconds.
- **Interoperable.** Any Lightning node that supports TAP can route asset payments.
- **Proven.** Tether launched USDt on Lightning using this exact stack.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                   Bolt Platform                     │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌────────────────┐   │
│  │  Mint /  │  │  Asset   │  │  Reserve &     │   │
│  │  Burn    │  │  Registry│  │  Proof System  │   │
│  │  Engine  │  │  (tapd)  │  │                │   │
│  └────┬─────┘  └────┬─────┘  └───────┬────────┘   │
│       │             │                │              │
│       └─────────────┴────────────────┘              │
│                     │                               │
│              ┌──────┴──────┐                        │
│              │  tapd (TAP  │                        │
│              │  Daemon)    │                        │
│              └──────┬──────┘                        │
│                     │                               │
│              ┌──────┴──────┐                        │
│              │  LND Node   │                        │
│              └──────┬──────┘                        │
└─────────────────────┼───────────────────────────────┘
                      │
              Bitcoin / Lightning Network
```

---

## Stack

| Layer | Technology |
|---|---|
| Asset Protocol | Taproot Assets Protocol (TAP) |
| Lightning Node | LND (Lightning Network Daemon) |
| Bitcoin Node | Bitcoin Core (full node) |
| Asset Daemon | tapd |
| Backend API | Go |
| Admin Dashboard | React / Next.js |
| Database | PostgreSQL |
| Infrastructure | Docker, Kubernetes |

---

## MVP: USD Stablecoin on Lightning

The first asset issued on Bolt will be **USDL** — a USD-pegged stablecoin on Lightning.

### MVP Scope
- Mint USDL backed by USD reserves
- Burn USDL on redemption
- Transfer USDL over Lightning channels
- Proof of reserves publication
- Admin dashboard for mint/burn operations
- Basic user wallet interface

---

## Roadmap & Implementation TODO

### Phase 0 — Foundation

#### Infrastructure
- [ ] Provision Bitcoin Core full node (mainnet + testnet instances)
- [ ] Provision LND node, configure with sufficient initial capacity
- [ ] Install and configure `tapd` (Taproot Assets daemon) linked to LND
- [ ] Set up watchtower node for channel monitoring
- [ ] Configure channel backup and SCB (Static Channel Backups) automation
- [ ] Set up persistent storage and database (PostgreSQL)
- [ ] Containerize all services with Docker Compose
- [ ] Set up Kubernetes manifests for production deployment
- [ ] Configure monitoring stack (Prometheus + Grafana dashboards for node health, channel liquidity, balance)
- [ ] Set up alerting (PagerDuty / Opsgenie) for node offline, low liquidity, mint anomalies
- [ ] Establish secrets management (Vault or AWS Secrets Manager) for node macaroons and seeds

#### Security Baseline
- [ ] Define key management architecture (HSM or cold-key multisig for reserve private keys)
- [ ] Set up macaroon-based RPC access controls for LND and tapd
- [ ] Implement IP allowlisting for node RPC endpoints
- [ ] Define seed phrase backup and disaster recovery procedures
- [ ] Conduct threat model for mint/burn operations

---

### Phase 1 — Asset Protocol Integration

#### Taproot Assets Setup
- [ ] Initialize tapd universe (public asset registry)
- [ ] Create USDL asset genesis on testnet
- [ ] Validate asset issuance and transfer on testnet
- [ ] Write tapd RPC wrapper client in Go (mint, burn, send, receive, asset info)
- [ ] Implement asset proof parsing and verification
- [ ] Test channel-based asset transfers end-to-end on testnet

#### Asset Registry
- [ ] Design asset metadata schema (name, ticker, decimals, asset ID, genesis point)
- [ ] Store and index asset issuance events in PostgreSQL
- [ ] Build asset proof archive and retrieval API
- [ ] Implement universe sync for public asset discovery

---

### Phase 2 — Mint & Burn Engine

#### Minting
- [ ] Design minting authorization model (multisig admin approval flow)
- [ ] Implement mint request workflow (request → approval → on-chain issuance)
- [ ] Integrate tapd `MintAsset` RPC into backend
- [ ] Emit mint event to audit log
- [ ] Enforce mint limits and rate limiting per authorized operator
- [ ] Generate and store Taproot Asset proof for each mint batch

#### Burning / Redemption
- [ ] Design burn/redemption request flow (user submits → admin verifies reserve → burns asset)
- [ ] Implement tapd burn (destroy asset supply)
- [ ] Track total supply change in database on every burn
- [ ] Emit burn event to audit log
- [ ] Generate burn proof and publish to universe

#### Supply Accounting
- [ ] Implement real-time total supply tracking (minted - burned)
- [ ] Reconcile on-chain asset proofs against internal ledger daily
- [ ] Build discrepancy detection and alerting

---

### Phase 3 — Reserve Management

#### Reserve System
- [ ] Define reserve architecture (custodian, bank account, or on-chain BTC/USD collateral)
- [ ] Build reserve balance API connector (Plaid / custodian API integration)
- [ ] Implement 1:1 reserve ratio enforcement before mint approval
- [ ] Automate reserve reconciliation (compare USDL total supply vs. USD reserve balance)
- [ ] Build proof-of-reserves publication pipeline (Merkle tree attestation)
- [ ] Publish signed proof-of-reserves report on a regular cadence (daily/weekly)

---

### Phase 4 — Backend API

#### Core API (Go / REST + gRPC)
- [ ] Design API schema (OpenAPI spec)
- [ ] Implement authentication (API keys + JWT for admin, macaroon-based for node ops)
- [ ] `POST /mint` — create mint request
- [ ] `POST /burn` — create burn/redemption request
- [ ] `GET /supply` — current total supply
- [ ] `GET /asset/:id` — asset metadata
- [ ] `GET /reserves` — current reserve attestation
- [ ] `GET /transactions` — paginated transfer history
- [ ] `POST /send` — initiate asset transfer over Lightning
- [ ] `POST /receive` — generate asset invoice
- [ ] Webhook system for payment/mint/burn event notifications
- [ ] Rate limiting and DDoS protection middleware
- [ ] Structured logging and request tracing (OpenTelemetry)

---

### Phase 5 — Admin Dashboard

#### Admin Portal (Next.js)
- [ ] Authentication with role-based access (super-admin, operator, auditor)
- [ ] Mint request queue — view pending requests, approve/reject
- [ ] Burn/redemption queue — view and process redemptions
- [ ] Supply overview — live total supply, mint/burn history chart
- [ ] Reserve dashboard — reserve balance vs. supply ratio gauge
- [ ] Node health panel — LND channel balances, tapd sync status
- [ ] Audit log viewer — all mint/burn/transfer events with timestamps and operator IDs
- [ ] Proof-of-reserves download page (signed attestation PDFs / JSON)

---

### Phase 6 — User Wallet Interface

#### End-User Wallet (React / Next.js or mobile)
- [ ] Connect wallet / onboarding flow
- [ ] Display USDL balance (on-chain asset + Lightning)
- [ ] Send USDL via Lightning invoice
- [ ] Receive USDL — generate Lightning invoice with asset metadata
- [ ] Transaction history with status and asset proof links
- [ ] Redemption flow (burn USDL, receive USD via bank/wire)
- [ ] QR code scan for payments

---

### Phase 7 — Compliance & Legal

- [ ] Research regulatory classification of USDL in target jurisdictions (US, EU)
- [ ] Engage legal counsel for money transmitter license requirements
- [ ] Implement KYC flow (identity verification via Persona, Jumio, or similar)
- [ ] Implement AML transaction monitoring (Chainalysis or TRM Labs integration)
- [ ] Implement sanctions screening (OFAC list check)
- [ ] Draft Terms of Service and Privacy Policy
- [ ] Define and document redemption eligibility rules
- [ ] Build SAR (Suspicious Activity Report) filing workflow
- [ ] Establish user data retention and deletion policies (GDPR / CCPA)

---

### Phase 8 — Testing

#### Unit & Integration Tests
- [ ] Unit tests for all mint/burn business logic
- [ ] Integration tests against tapd on regtest
- [ ] Integration tests for LND channel open/close/rebalance scenarios
- [ ] End-to-end test: mint → transfer over Lightning → burn
- [ ] Reserve reconciliation tests with mock custodian API

#### Security Testing
- [ ] Penetration test on admin API and dashboard
- [ ] Smart contract / protocol audit (external firm)
- [ ] Fuzz test asset proof parser
- [ ] Channel force-close and griefing attack simulations
- [ ] Key compromise recovery drill

#### Load & Reliability Testing
- [ ] Load test Lightning payment throughput (target TPS for MVP)
- [ ] Simulate channel liquidity exhaustion and rebalancing under load
- [ ] Chaos engineering: node restart, database failover, network partition

---

### Phase 9 — Launch Preparation

- [ ] Testnet public beta with invited participants
- [ ] Security audit remediation
- [ ] Mainnet deployment runbook
- [ ] Incident response playbook
- [ ] Node capacity planning (initial channel capacity in BTC)
- [ ] Liquidity provider agreements (LSP partnerships for routing)
- [ ] Press / announcement preparation
- [ ] Public universe registration of USDL asset genesis

---

## Getting Started (Development)

```bash
# Clone the repo
git clone https://github.com/houseofhufflepuff/bolt.git
cd bolt

# Start local stack (Bitcoin regtest + LND + tapd)
docker compose up -d

# Initialize tapd universe
./scripts/init-universe.sh

# Run tests
make test
```

> Full setup guide: [docs/setup.md](docs/setup.md) _(coming soon)_

---

## References

- [Taproot Assets Protocol — Lightning Labs](https://docs.lightning.engineering/the-lightning-network/taproot-assets)
- [tapd GitHub](https://github.com/lightninglabs/taproot-assets)
- [LND Documentation](https://docs.lightning.engineering/lightning-network-tools/lnd)
- [Tether on Lightning](https://tether.to/en/tether-launches-usdt-on-lightning-network/)
- [Bitcoin Improvement Proposals — Taproot (BIP 341)](https://github.com/bitcoin/bips/blob/master/bip-0341.mediawiki)

---

## License

MIT
