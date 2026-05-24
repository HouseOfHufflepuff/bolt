BIN         := bolt
TAPD_TAG    ?= v0.4.0-alpha
LND_TAG     ?= v0.18.0-beta
PROTO_TMP   := .proto-tmp
GOOGLEAPIS  := $(PROTO_TMP)/googleapis

# Runtime config — override via environment or .env
API_URL     ?= http://localhost:8080
API_KEY     ?= dev-key
SYMBOL      ?= USDL
AMOUNT      ?= 1000000
OPERATOR    ?= admin
DEST        ?= ""

# --- Build ---

build:
	go build -o $(BIN) ./cmd/server

run:
	BOLT_API_KEY=$(API_KEY) go run ./cmd/server

# --- Docker stack ---

up:
	docker compose up -d

down:
	docker compose down

# --- Database ---

db-init:
	cockroach sql --insecure --host=localhost:26257 -e "CREATE DATABASE IF NOT EXISTS bolt"

migrate:
	goose -dir db/migrations postgres "$$BOLT_DB_URL" up

# --- Platform commands (server must be running: make run) ---

# Check server health
health:
	curl -sf $(API_URL)/healthz && echo " OK"

# GET current supply of all assets
# Usage: make supply
supply:
	curl -s $(API_URL)/supply | jq .

# GET current oracle prices + BOLT NAV
# Usage: make price
price:
	curl -s $(API_URL)/price | jq .

# GET basket weights + current drift
# Usage: make weights
weights:
	curl -s $(API_URL)/weights | jq .

# GET transaction history for an asset
# Usage: make transactions SYMBOL=USDL
# Usage: make transactions SYMBOL=BOLT
transactions:
	curl -s "$(API_URL)/transactions?symbol=$(SYMBOL)&limit=20" | jq .

# POST mint a constituent asset (requires tapd)
# Usage: make mint SYMBOL=USDL AMOUNT=1000000 OPERATOR=admin
mint:
	curl -s -X POST $(API_URL)/mint \
	  -H "X-API-Key: $(API_KEY)" \
	  -H "Content-Type: application/json" \
	  -d '{"asset_symbol":"$(SYMBOL)","amount":$(AMOUNT),"operator":"$(OPERATOR)"}' | jq .

# POST mint BOLT tokens from constituent deposits (requires tapd)
# Usage: make mint-bolt USDL=500000 CHFL=300000 NOKL=100000 BTC=1540
USDL ?= 500000
CHFL ?= 300000
NOKL ?= 100000
BTC  ?= 1540
mint-bolt:
	curl -s -X POST $(API_URL)/mint/bolt \
	  -H "X-API-Key: $(API_KEY)" \
	  -H "Content-Type: application/json" \
	  -d '{"usdl":$(USDL),"chfl":$(CHFL),"nokl":$(NOKL),"btc":$(BTC),"operator":"$(OPERATOR)"}' | jq .

# POST burn tokens (requires tapd)
# Usage: make burn SYMBOL=USDL AMOUNT=500000 OPERATOR=admin
burn:
	curl -s -X POST $(API_URL)/burn \
	  -H "X-API-Key: $(API_KEY)" \
	  -H "Content-Type: application/json" \
	  -d '{"asset_symbol":"$(SYMBOL)","amount":$(AMOUNT),"operator":"$(OPERATOR)"}' | jq .

# POST redeem an asset (BTC requires DEST=lnbc... payment request; fiat is stubbed)
# Usage: make redeem SYMBOL=BTC AMOUNT=1000 DEST=lnbc1...
# Usage: make redeem SYMBOL=USDL AMOUNT=500000
redeem:
	curl -s -X POST $(API_URL)/redeem/$(SYMBOL) \
	  -H "X-API-Key: $(API_KEY)" \
	  -H "Content-Type: application/json" \
	  -d '{"amount":$(AMOUNT),"operator":"$(OPERATOR)","destination":"$(DEST)"}' | jq .

# GET reserve snapshot + signed Merkle proof (requires LND for BTC balance)
# Usage: make reserves
reserves:
	curl -s $(API_URL)/reserves \
	  -H "X-API-Key: $(API_KEY)" | jq .

# POST trigger rebalance (computes drift, logs trade queue)
# Usage: make rebalance
rebalance:
	curl -s -X POST $(API_URL)/rebalance \
	  -H "X-API-Key: $(API_KEY)" | jq .

# --- Proto generation ---

proto-gen: proto-check
	@rm -rf $(PROTO_TMP)
	@mkdir -p \
		$(PROTO_TMP)/tapd/taprpc \
		$(PROTO_TMP)/tapd/universerpc \
		$(PROTO_TMP)/lnd/lnrpc \
		$(PROTO_TMP)/lnd/routerrpc \
		$(PROTO_TMP)/lnd/walletrpc \
		$(GOOGLEAPIS)/google/api \
		internal/tapd/pb \
		internal/lnd/pb
	@echo "fetching proto files..."
	@curl -sSL https://raw.githubusercontent.com/lightninglabs/taproot-assets/$(TAPD_TAG)/taprpc/tap.proto \
		-o $(PROTO_TMP)/tapd/taprpc/tap.proto
	@curl -sSL https://raw.githubusercontent.com/lightninglabs/taproot-assets/$(TAPD_TAG)/universerpc/universe.proto \
		-o $(PROTO_TMP)/tapd/universerpc/universe.proto
	@curl -sSL https://raw.githubusercontent.com/lightningnetwork/lnd/$(LND_TAG)/lnrpc/rpc.proto \
		-o $(PROTO_TMP)/lnd/lnrpc/rpc.proto
	@curl -sSL https://raw.githubusercontent.com/lightningnetwork/lnd/$(LND_TAG)/routerrpc/router.proto \
		-o $(PROTO_TMP)/lnd/routerrpc/router.proto
	@curl -sSL https://raw.githubusercontent.com/lightningnetwork/lnd/$(LND_TAG)/walletrpc/walletkit.proto \
		-o $(PROTO_TMP)/lnd/walletrpc/walletkit.proto
	@curl -sSL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto \
		-o $(GOOGLEAPIS)/google/api/annotations.proto
	@curl -sSL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto \
		-o $(GOOGLEAPIS)/google/api/http.proto
	@echo "generating tapd stubs..."
	protoc -I$(PROTO_TMP)/tapd -I$(GOOGLEAPIS) \
		--go_out=internal/tapd/pb --go_opt=paths=source_relative \
		--go-grpc_out=internal/tapd/pb --go-grpc_opt=paths=source_relative \
		$(PROTO_TMP)/tapd/taprpc/tap.proto \
		$(PROTO_TMP)/tapd/universerpc/universe.proto
	@echo "generating lnd stubs..."
	protoc -I$(PROTO_TMP)/lnd -I$(GOOGLEAPIS) \
		--go_out=internal/lnd/pb --go_opt=paths=source_relative \
		--go-grpc_out=internal/lnd/pb --go-grpc_opt=paths=source_relative \
		$(PROTO_TMP)/lnd/lnrpc/rpc.proto \
		$(PROTO_TMP)/lnd/routerrpc/router.proto \
		$(PROTO_TMP)/lnd/walletrpc/walletkit.proto
	@rm -rf $(PROTO_TMP)
	@echo "done"

proto-check:
	@which protoc >/dev/null 2>&1 || (echo "install protoc: brew install protobuf" && exit 1)
	@which protoc-gen-go >/dev/null 2>&1 || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@which protoc-gen-go-grpc >/dev/null 2>&1 || go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# --- Housekeeping ---

clean:
	rm -f $(BIN)
	rm -rf $(PROTO_TMP)

.PHONY: build run up down db-init migrate \
        health supply price weights transactions \
        mint mint-bolt burn redeem reserves rebalance \
        proto-gen proto-check clean
