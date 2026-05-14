# Kurtosis URL helpers for ethpandaops/ethereum-package devnets.
# Source from sibling scripts:  . "$(dirname "$0")/env.sh"
#
# Port IDs match package constants (see ethereum-package src/package_io/constants.star):
#   CL beacon REST:  http
#   EL JSON-RPC:     rpc
#
# Service names include a UUID prefix and depend on package version. Discover with:
#   kurtosis enclave inspect "$KURTOSIS_ENCLAVE"
# Then set overrides, for example:
#   export KURTOSIS_CL_SERVICE='cl-1-lighthouse-...'
#   export KURTOSIS_EL_SERVICE='el-1-geth-...'

: "${KURTOSIS_ENCLAVE:=pauli-dev-network}"

# Optional overrides (empty = try auto-detect)
: "${KURTOSIS_CL_SERVICE:=}"
: "${KURTOSIS_EL_SERVICE:=}"

: "${KURTOSIS_CL_HTTP_PORT_ID:=http}"
: "${KURTOSIS_EL_RPC_PORT_ID:=rpc}"

kurtosis_require_cli() {
  command -v kurtosis >/dev/null 2>&1 || {
    echo "kurtosis: command not found; install https://docs.kurtosis.com/install" >&2
    return 1
  }
}

# Print published host:port for a service (Kurtosis CLI output is typically "127.0.0.1:PORT").
kurtosis_port_addr() {
  local service="$1"
  local port_id="$2"
  kurtosis_require_cli || return 1
  if [[ -z "$service" ]]; then
    echo "kurtosis_port_addr: service name is empty" >&2
    return 1
  fi
  kurtosis port print "$KURTOSIS_ENCLAVE" "$service" "$port_id" 2>/dev/null | tr -d '\r'
}

# Prefix http:// if the addr has no scheme.
_http_url_from_addr() {
  local addr="$1"
  if [[ "$addr" == http://* || "$addr" == https://* ]]; then
    echo "$addr"
  else
    echo "http://${addr}"
  fi
}

# Auto-pick first service token from `kurtosis enclave inspect` text (names look like
# cl-1-<uuid>-lighthouse-geth). Best-effort; override KURTOSIS_*_SERVICE when multiple participants exist.
_kurtosis_first_service_token() {
  local prefix_re="$1"
  kurtosis_require_cli || return 1
  kurtosis enclave inspect "$KURTOSIS_ENCLAVE" 2>/dev/null | grep -oE "${prefix_re}" | head -1
}

kurtosis_resolve_cl_service() {
  if [[ -n "${KURTOSIS_CL_SERVICE}" ]]; then
    echo "${KURTOSIS_CL_SERVICE}"
    return
  fi
  local s
  s=$(_kurtosis_first_service_token 'cl-[0-9]+-[[:alnum:]]+-lighthouse-geth') || true
  if [[ -z "$s" ]]; then
    s=$(_kurtosis_first_service_token 'cl-[0-9]+-[[:alnum:]-]+') || true
  fi
  if [[ -z "$s" ]]; then
    echo "Set KURTOSIS_CL_SERVICE to your beacon node service name (see: kurtosis enclave inspect $KURTOSIS_ENCLAVE)" >&2
    return 1
  fi
  echo "$s"
}

kurtosis_resolve_el_service() {
  if [[ -n "${KURTOSIS_EL_SERVICE}" ]]; then
    echo "${KURTOSIS_EL_SERVICE}"
    return
  fi
  local s
  s=$(_kurtosis_first_service_token 'el-[0-9]+-[[:alnum:]]+-geth') || true
  if [[ -z "$s" ]]; then
    s=$(_kurtosis_first_service_token 'el-[0-9]+-[[:alnum:]-]+') || true
  fi
  if [[ -z "$s" ]]; then
    echo "Set KURTOSIS_EL_SERVICE to your execution client service name (see: kurtosis enclave inspect $KURTOSIS_ENCLAVE)" >&2
    return 1
  fi
  echo "$s"
}

# Beacon HTTP API base URL (no trailing slash).
beacon_http_url() {
  local svc addr
  svc=$(kurtosis_resolve_cl_service) || return 1
  addr=$(kurtosis_port_addr "$svc" "$KURTOSIS_CL_HTTP_PORT_ID") || return 1
  _http_url_from_addr "$addr"
}

# Execution JSON-RPC base URL (no trailing slash).
el_rpc_url() {
  local svc addr
  svc=$(kurtosis_resolve_el_service) || return 1
  addr=$(kurtosis_port_addr "$svc" "$KURTOSIS_EL_RPC_PORT_ID") || return 1
  _http_url_from_addr "$addr"
}

export_beacon_and_el_urls() {
  export BEACON_HTTP_URL
  export EL_RPC_URL
  BEACON_HTTP_URL=$(beacon_http_url) || return 1
  EL_RPC_URL=$(el_rpc_url) || return 1
  echo "BEACON_HTTP_URL=$BEACON_HTTP_URL"
  echo "EL_RPC_URL=$EL_RPC_URL"
}
