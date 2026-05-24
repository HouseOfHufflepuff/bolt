#!/usr/bin/env bash
set -euo pipefail

BITCOIN="docker exec -T bolt-bitcoind bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin"
LND="docker exec -T bolt-lnd lncli --network=regtest"

echo "waiting for LND wallet..."
until $LND getinfo >/dev/null 2>&1; do
  echo "  lnd not ready, retrying in 3s..."
  sleep 3
done
echo "LND ready"

ADDR=$($LND newaddress p2wkh | jq -r .address)
echo "LND address: $ADDR"

echo "mining 101 blocks for coinbase maturity..."
$BITCOIN generatetoaddress 101 "$ADDR"

echo ""
echo "wallet balance:"
$LND walletbalance
