#!/usr/bin/env bash
# Send repeating self-transfers on the Kurtosis EL with a non-zero priority fee so blocks
# carry execution-layer fee income. Requires cast (Foundry) and PRIVATE_KEY in the environment.
#
# Usage:
#   export PRIVATE_KEY=0x...   # prefunded dev account; never commit keys
#   ./scripts/kurtosis/spam-tx.sh
# Optional:
#   EL_RPC_URL=... BEACON_HTTP_URL=... ./scripts/kurtosis/spam-tx.sh
#   SPAM_INTERVAL_SEC=0.2 PRIORITY_GAS_PRICE_GWEI=2 SPAM_COUNT=0 ./scripts/kurtosis/spam-tx.sh
#   SPAM_COUNT=0 means infinite loop (default).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=env.sh
. "${SCRIPT_DIR}/env.sh"

: "${SPAM_INTERVAL_SEC:=0.25}"
: "${PRIORITY_GAS_PRICE_GWEI:=2}"
: "${SPAM_COUNT:=0}"

command -v cast >/dev/null 2>&1 || {
  echo "cast not found; install Foundry https://book.getfoundry.sh/getting-started/installation" >&2
  exit 1
}

if [[ -z "${PRIVATE_KEY:-}" ]]; then
  echo "PRIVATE_KEY must be set (hex with 0x prefix) to a prefunded account on this devnet." >&2
  echo "Use genesis / mnemonic from your ethereum-package run output; never commit keys." >&2
  exit 1
fi

if [[ -z "${EL_RPC_URL:-}" ]]; then
  EL_RPC_URL=$(el_rpc_url) || exit 1
fi

FROM_ADDR=$(cast wallet address "${PRIVATE_KEY}")
echo "EL_RPC_URL=${EL_RPC_URL}"
echo "Sending from ${FROM_ADDR} every ${SPAM_INTERVAL_SEC}s (priority ${PRIORITY_GAS_PRICE_GWEI} gwei)"

i=0
while true; do
  cast send --rpc-url "${EL_RPC_URL}" \
    --private-key "${PRIVATE_KEY}" \
    --priority-gas-price "${PRIORITY_GAS_PRICE_GWEI}gwei" \
    --gas-price "$(( PRIORITY_GAS_PRICE_GWEI + 1 ))gwei" \
    "${FROM_ADDR}" \
    --value 0 >/dev/null
  i=$((i + 1))
  if [[ "${SPAM_COUNT}" -gt 0 && "${i}" -ge "${SPAM_COUNT}" ]]; then
    break
  fi
  sleep "${SPAM_INTERVAL_SEC}"
done
