#!/usr/bin/env bash
set -euo pipefail

TAPD="docker exec -T bolt-tapd tapcli --network=regtest"
LND="docker exec -T bolt-lnd lncli --network=regtest"
BITCOIN="docker exec -T bolt-bitcoind bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin"
CRDB="docker exec -T bolt-cockroachdb cockroach sql --insecure --host=localhost:26257 --database=bolt"

echo "waiting for tapd..."
until $TAPD getinfo >/dev/null 2>&1; do
  echo "  tapd not ready, retrying in 3s..."
  sleep 3
done
echo "tapd ready"

echo "minting genesis assets (grouped for reissuance)..."
$TAPD assets mint --type normal --name USDL --supply 10000000 --new_grouped_asset
$TAPD assets mint --type normal --name CHFL --supply 10000000 --new_grouped_asset
$TAPD assets mint --type normal --name NOKL --supply 10000000 --new_grouped_asset
$TAPD assets mint --type normal --name BOLT --supply 10000000 --new_grouped_asset

echo "finalizing batch..."
$TAPD assets mint finalize

MINE_ADDR=$($LND newaddress p2wkh | jq -r .address)
echo "mining 1 block to confirm assets..."
$BITCOIN generatetoaddress 1 "$MINE_ADDR"

echo "waiting for asset confirmation..."
sleep 5

echo "seeding asset records in CockroachDB..."
$CRDB -e "
INSERT INTO assets (symbol, total_supply)
VALUES ('USDL', 0), ('CHFL', 0), ('NOKL', 0), ('BOLT', 0)
ON CONFLICT (symbol) DO NOTHING;"

echo ""
echo "tapd assets:"
$TAPD assets list | jq '[.assets[] | {name: .asset_genesis.name, supply: .amount}]'

echo ""
echo "db assets:"
$CRDB -e "SELECT symbol, total_supply FROM assets ORDER BY symbol;"
