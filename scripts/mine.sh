#!/usr/bin/env bash
set -euo pipefail

N=${1:-1}

ADDR=$(docker exec -T bolt-lnd lncli --network=regtest newaddress p2wkh | jq -r .address)
docker exec -T bolt-bitcoind bitcoin-cli \
  -regtest -rpcuser=bitcoin -rpcpassword=bitcoin \
  generatetoaddress "$N" "$ADDR"

echo "mined $N block(s) → $ADDR"
