#!/usr/bin/env bash
# For a given validator index, find its proposer duties in the current and next epoch,
# then send priority-fee txs around each assigned slot wall time (fast devnets).
#
# Usage:
#   export PRIVATE_KEY=0x...
#   ./scripts/kurtosis/burst-around-proposer.sh 5
# Env:
#   BEACON_HTTP_URL — optional; resolved via env.sh if unset
#   EL_RPC_URL      — optional; resolved via env.sh if unset
#   BURST_LEAD_SEC  seconds before slot time to start spam (default 1)
#   BURST_TAIL_SEC  seconds after slot time to stop (default 4)
#   KURTOSIS_*      see env.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=env.sh
. "${SCRIPT_DIR}/env.sh"

: "${BURST_LEAD_SEC:=1}"
: "${BURST_TAIL_SEC:=4}"
: "${SLOTS_PER_EPOCH:=32}"
: "${SPAM_INTERVAL_SEC:=0.15}"
: "${PRIORITY_GAS_PRICE_GWEI:=2}"

if [[ "${#}" -lt 1 ]]; then
  echo "usage: $0 <validator_index>" >&2
  exit 1
fi

VALIDATOR_INDEX="$1"

command -v cast >/dev/null 2>&1 || {
  echo "cast not found; install Foundry" >&2
  exit 1
}
command -v curl >/dev/null 2>&1 || {
  echo "curl is required" >&2
  exit 1
}
command -v jq >/dev/null 2>&1 || {
  echo "jq is required" >&2
  exit 1
}

if [[ -z "${PRIVATE_KEY:-}" ]]; then
  echo "PRIVATE_KEY must be set (see spam-tx.sh)" >&2
  exit 1
fi

if [[ -z "${BEACON_HTTP_URL:-}" ]]; then
  BEACON_HTTP_URL=$(beacon_http_url) || exit 1
fi
if [[ -z "${EL_RPC_URL:-}" ]]; then
  EL_RPC_URL=$(el_rpc_url) || exit 1
fi

FROM_ADDR=$(cast wallet address "${PRIVATE_KEY}")

head_slot=$(curl -fsS "${BEACON_HTTP_URL}/eth/v1/beacon/headers/head" | jq -r '.data.header.message.slot | tonumber')
genesis_time=$(curl -fsS "${BEACON_HTTP_URL}/eth/v1/beacon/genesis" | jq -r '.data.genesis_time | tonumber')
slot_duration=$(curl -fsS "${BEACON_HTTP_URL}/eth/v1/config/spec" | jq -r '.data.SECONDS_PER_SLOT | tonumber')

echo "BEACON_HTTP_URL=${BEACON_HTTP_URL}"
echo "EL_RPC_URL=${EL_RPC_URL}"
echo "validator=${VALIDATOR_INDEX} head_slot=${head_slot} slot_duration=${slot_duration}s"

slots_json() {
  local ep="$1"
  curl -fsS -X POST "${BEACON_HTTP_URL}/eth/v1/validator/duties/proposer/${ep}" \
    -H 'Content-Type: application/json' \
    -d "[\"${VALIDATOR_INDEX}\"]"
}

d1=$(slots_json "${head_epoch}")
d2=$(slots_json "$((head_epoch + 1))")
merged=$(printf '%s\n%s\n' "${d1}" "${d2}" | jq -s '{data: (.[0].data + .[1].data)}')

slots=()
while IFS= read -r s; do
  [[ -n "${s}" ]] && slots+=("$s")
done < <(jq -r --argjson v "${VALIDATOR_INDEX}" '
  .data[]
  | select((.validator_index|tonumber) == $v)
  | .slot | tonumber
' <<<"${merged}" | sort -n | uniq)

if [[ "${#slots[@]}" -eq 0 ]]; then
  echo "No proposer duties found for validator ${VALIDATOR_INDEX} in epochs ${head_epoch} or $((head_epoch + 1))." >&2
  exit 1
fi

slot_time_unix() {
  local slot="$1"
  echo $((genesis_time + slot * slot_duration))
}

echo "Upcoming proposer slots for this validator: ${slots[*]}"

burst_send() {
  local burst_end="$1"
  while [[ $(date +%s) -lt "${burst_end}" ]]; do
    cast send --rpc-url "${EL_RPC_URL}" \
      --private-key "${PRIVATE_KEY}" \
      --priority-gas-price "${PRIORITY_GAS_PRICE_GWEI}gwei" \
      --gas-price "$(( PRIORITY_GAS_PRICE_GWEI + 1 ))gwei" \
      "${FROM_ADDR}" \
      --value 0 >/dev/null 2>&1 || true
    sleep "${SPAM_INTERVAL_SEC}"
  done
}

for slot in "${slots[@]}"; do
  t=$(slot_time_unix "${slot}")
  now=$(date +%s)
  start=$((t - BURST_LEAD_SEC))
  end=$((t + BURST_TAIL_SEC))
  if [[ "${end}" -lt "${now}" ]]; then
    continue
  fi
  if [[ "${start}" -gt "${now}" ]]; then
    sleep $((start - now))
  fi
  echo "--- burst for proposer slot ${slot} (slot unix ${t}) ---"
  burst_send "${end}"
done

echo "Done."
