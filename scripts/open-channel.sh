#!/usr/bin/env bash
set -euo pipefail

LND="docker exec -T bolt-lnd lncli --network=regtest"
LNDR="docker exec -T bolt-lnd-receiver lncli --network=regtest"
BITCOIN="docker exec -T bolt-bitcoind bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin"

echo "waiting for lnd-receiver..."
until $LNDR getinfo >/dev/null 2>&1; do
  echo "  lnd-receiver not ready, retrying in 3s..."
  sleep 3
done

RECV_PUBKEY=$($LNDR getinfo | jq -r .identity_pubkey)
echo "receiver pubkey: $RECV_PUBKEY"

echo "connecting peers..."
$LND connect "${RECV_PUBKEY}@lnd-receiver:9735" 2>/dev/null || echo "  (already connected or connect queued)"

echo "opening 5M sat channel with 2.5M push..."
$LND openchannel \
  --node_key="$RECV_PUBKEY" \
  --local_amt=5000000 \
  --push_amt=2500000

MINE_ADDR=$($LND newaddress p2wkh | jq -r .address)
echo "mining 3 confirmation blocks..."
$BITCOIN generatetoaddress 3 "$MINE_ADDR"

echo ""
echo "channels:"
$LND listchannels | jq '.channels[] | {remote_pubkey, capacity, local_balance, remote_balance}'
