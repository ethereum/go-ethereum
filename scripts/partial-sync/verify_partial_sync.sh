#!/usr/bin/env bash
#
# verify_partial_sync.sh - Verify partial state sync correctness.
#
# Runs JSON-RPC checks against a running geth node to verify:
# 1. All accounts are accessible (full account trie synced)
# 2. Tracked contract storage and code are present
# 3. Untracked contract storage and code are correctly rejected
#
# Usage:
#   ./verify_partial_sync.sh              # RPC checks (geth must be running)
#   ./verify_partial_sync.sh --db-only    # Database inspection (geth must be stopped)
#   ./verify_partial_sync.sh --all        # Both (stops geth for DB checks)
#
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
DATADIR="${DATADIR:-$HOME/.ethereum-partial-test}"
GETH="${GETH:-$(dirname "${BASH_SOURCE[0]}")/../../build/bin/geth}"

# Tracked contracts (WETH, DAI)
WETH="0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
DAI="0x6B175474E89094C44Da98b954EedeAC495271d0F"

# Untracked contracts (USDC, Uniswap V2 Router)
USDC="0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
UNISWAP_ROUTER="0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"

# ERC20 totalSupply() selector
TOTAL_SUPPLY="0x18160ddd"

# Counters
PASS=0
FAIL=0
TOTAL=0

# ─── Helpers ──────────────────────────────────────────────────────────

check_deps() {
    for cmd in curl jq; do
        if ! command -v "$cmd" &>/dev/null; then
            echo "ERROR: '$cmd' is required but not installed."
            exit 1
        fi
    done
}

rpc_call() {
    local method="$1"
    local params="$2"
    curl -s -X POST "$RPC_URL" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"params\":$params,\"id\":1}"
}

# Check that result field is non-zero hex
check_nonzero() {
    local label="$1"
    local method="$2"
    local params="$3"

    TOTAL=$((TOTAL + 1))
    local response
    response=$(rpc_call "$method" "$params")

    local error
    error=$(echo "$response" | jq -r '.error // empty')
    if [ -n "$error" ]; then
        echo "  [FAIL] $label"
        echo "         Error: $(echo "$response" | jq -r '.error.message')"
        FAIL=$((FAIL + 1))
        return
    fi

    local result
    result=$(echo "$response" | jq -r '.result')

    if [ "$result" = "0x0" ] || [ "$result" = "0x" ] || [ "$result" = "null" ] || [ -z "$result" ]; then
        echo "  [FAIL] $label (got: $result)"
        FAIL=$((FAIL + 1))
    else
        # Truncate long results for display
        local display="$result"
        if [ ${#display} -gt 20 ]; then
            display="${display:0:20}..."
        fi
        echo "  [PASS] $label ($display)"
        PASS=$((PASS + 1))
    fi
}

# Check that result is non-empty bytecode (not "0x")
check_code() {
    local label="$1"
    local addr="$2"

    TOTAL=$((TOTAL + 1))
    local response
    response=$(rpc_call "eth_getCode" "[\"$addr\",\"latest\"]")

    local error
    error=$(echo "$response" | jq -r '.error // empty')
    if [ -n "$error" ]; then
        echo "  [FAIL] $label"
        echo "         Error: $(echo "$response" | jq -r '.error.message')"
        FAIL=$((FAIL + 1))
        return
    fi

    local result
    result=$(echo "$response" | jq -r '.result')
    local len=$(( (${#result} - 2) / 2 ))  # bytes = (hex_len - "0x" prefix) / 2

    if [ "$result" = "0x" ] || [ "$len" -le 0 ]; then
        echo "  [FAIL] $label (empty code)"
        FAIL=$((FAIL + 1))
    else
        echo "  [PASS] $label ($len bytes)"
        PASS=$((PASS + 1))
    fi
}

# Check that RPC returns a specific error code
check_error() {
    local label="$1"
    local method="$2"
    local params="$3"
    local expected_code="$4"

    TOTAL=$((TOTAL + 1))
    local response
    response=$(rpc_call "$method" "$params")

    local error_code
    error_code=$(echo "$response" | jq -r '.error.code // empty')

    if [ "$error_code" = "$expected_code" ]; then
        local msg
        msg=$(echo "$response" | jq -r '.error.message')
        echo "  [PASS] $label (error $error_code: $msg)"
        PASS=$((PASS + 1))
    elif [ -n "$error_code" ]; then
        echo "  [FAIL] $label (expected error $expected_code, got $error_code)"
        FAIL=$((FAIL + 1))
    else
        local result
        result=$(echo "$response" | jq -r '.result')
        echo "  [FAIL] $label (expected error $expected_code, but got result: ${result:0:20}...)"
        FAIL=$((FAIL + 1))
    fi
}

# Check that eth_call returns an error (any error)
check_call_error() {
    local label="$1"
    local to="$2"
    local data="$3"

    TOTAL=$((TOTAL + 1))
    local response
    response=$(rpc_call "eth_call" "[{\"to\":\"$to\",\"data\":\"$data\"},\"latest\"]")

    local error
    error=$(echo "$response" | jq -r '.error // empty')

    if [ -n "$error" ]; then
        local msg
        msg=$(echo "$response" | jq -r '.error.message')
        echo "  [PASS] $label (error: ${msg:0:50})"
        PASS=$((PASS + 1))
    else
        local result
        result=$(echo "$response" | jq -r '.result')
        echo "  [FAIL] $label (expected error, got result: ${result:0:20}...)"
        FAIL=$((FAIL + 1))
    fi
}

# ─── RPC Verification ────────────────────────────────────────────────

run_rpc_checks() {
    echo "=== Partial State Sync Verification ==="
    echo ""
    echo "RPC endpoint: $RPC_URL"
    echo ""

    # A. Sync Status
    echo "Sync Status:"

    TOTAL=$((TOTAL + 1))
    local syncing
    syncing=$(rpc_call "eth_syncing" "[]" | jq -r '.result')
    if [ "$syncing" = "false" ]; then
        echo "  [PASS] eth_syncing returns false"
        PASS=$((PASS + 1))
    else
        echo "  [WARN] eth_syncing returns: $syncing (sync may still be in progress)"
        echo "         Some checks may fail until sync completes."
        PASS=$((PASS + 1))  # Not a failure, just a warning
    fi

    TOTAL=$((TOTAL + 1))
    local block_hex
    block_hex=$(rpc_call "eth_blockNumber" "[]" | jq -r '.result')
    if [ -n "$block_hex" ] && [ "$block_hex" != "null" ]; then
        local block_dec
        block_dec=$(printf "%d" "$block_hex" 2>/dev/null || echo "?")
        echo "  [PASS] Block number: $block_dec ($block_hex)"
        PASS=$((PASS + 1))
    else
        echo "  [FAIL] Could not get block number"
        FAIL=$((FAIL + 1))
    fi
    echo ""

    # B. Account Data (all accounts - full trie synced)
    echo "Account Data (all accounts - full trie synced):"
    check_nonzero "USDC contract balance" "eth_getBalance" "[\"$USDC\",\"latest\"]"
    check_nonzero "WETH contract balance" "eth_getBalance" "[\"$WETH\",\"latest\"]"
    check_nonzero "Uniswap Router balance" "eth_getBalance" "[\"$UNISWAP_ROUTER\",\"latest\"]"
    check_nonzero "USDC nonce" "eth_getTransactionCount" "[\"$USDC\",\"latest\"]"
    echo ""

    # C. Tracked Contracts (WETH, DAI)
    echo "Tracked Contracts (WETH, DAI):"
    check_code "WETH code" "$WETH"
    check_code "DAI code" "$DAI"
    check_nonzero "WETH storage slot 0x0" "eth_getStorageAt" "[\"$WETH\",\"0x0\",\"latest\"]"
    check_nonzero "DAI storage slot 0x0" "eth_getStorageAt" "[\"$DAI\",\"0x0\",\"latest\"]"
    check_nonzero "eth_call WETH.totalSupply()" "eth_call" "[{\"to\":\"$WETH\",\"data\":\"$TOTAL_SUPPLY\"},\"latest\"]"
    check_nonzero "eth_call DAI.totalSupply()" "eth_call" "[{\"to\":\"$DAI\",\"data\":\"$TOTAL_SUPPLY\"},\"latest\"]"
    echo ""

    # D. Untracked Contracts (USDC, Uniswap V2 Router)
    echo "Untracked Contracts (USDC, Uniswap V2 Router):"
    check_error "USDC eth_getStorageAt" "eth_getStorageAt" "[\"$USDC\",\"0x0\",\"latest\"]" "-32001"
    check_error "Router eth_getStorageAt" "eth_getStorageAt" "[\"$UNISWAP_ROUTER\",\"0x0\",\"latest\"]" "-32001"
    check_error "USDC eth_getCode" "eth_getCode" "[\"$USDC\",\"latest\"]" "-32002"
    check_error "Router eth_getCode" "eth_getCode" "[\"$UNISWAP_ROUTER\",\"latest\"]" "-32002"
    check_call_error "eth_call USDC.totalSupply()" "$USDC" "$TOTAL_SUPPLY"
    echo ""

    # Summary
    echo "========================================="
    if [ $FAIL -eq 0 ]; then
        echo "  Results: $PASS/$TOTAL passed"
    else
        echo "  Results: $PASS/$TOTAL passed, $FAIL FAILED"
    fi
    echo "========================================="
}

# ─── Database Verification ───────────────────────────────────────────

run_db_checks() {
    echo ""
    echo "=== Database-Level Verification ==="
    echo ""
    echo "Data directory: $DATADIR"
    echo ""

    # Check geth binary exists
    if [ ! -x "$GETH" ]; then
        echo "ERROR: geth binary not found at $GETH"
        echo "Set GETH env var or build first: go build -o build/bin/geth ./cmd/geth"
        exit 1
    fi

    # Check datadir exists
    if [ ! -d "$DATADIR" ]; then
        echo "ERROR: Data directory not found: $DATADIR"
        exit 1
    fi

    # Check geth is not running (LevelDB requires exclusive access)
    if pgrep -f "geth.*partial-test" > /dev/null 2>&1; then
        echo "WARNING: geth appears to be running. Stop it first for database inspection."
        echo "  kill \$(pgrep -f 'geth.*partial-test')"
        echo ""
    fi

    echo "Running: geth db inspect"
    echo "(this may take a while for large databases)"
    echo ""

    "$GETH" db inspect --datadir "$DATADIR" 2>&1 | tee /tmp/partial-sync-inspect.txt

    echo ""
    echo "Inspection output saved to: /tmp/partial-sync-inspect.txt"
    echo ""
    echo "What to check in the output above:"
    echo "  - 'Account snapshot'  : Should be large (~45 GiB) - full account trie"
    echo "  - 'Storage snapshot'  : Should be TINY (< 1 GiB) - only WETH + DAI"
    echo "  - 'Contract codes'    : Should be very small - only 2 contracts"
    echo "  - 'Bodies'            : Should be tiny (< 10 MiB) - chain retention=1024"
    echo "  - 'Receipts'          : Should be tiny (< 10 MiB) - chain retention=1024"
    echo "  - 'Headers'           : ~9 GiB (full chain, non-prunable)"
    echo "  - Compare total DB size to a full node (~640+ GiB)"
    echo "  - Expected total: ~59 GiB (headers + partial state)"
    echo ""

    # Try dumptrie for tracked contract (WETH)
    echo "Verifying tracked contract storage (WETH)..."
    echo "Running: geth db dumptrie (limited to 5 entries)"
    echo ""

    # Compute WETH account hash (keccak256 of address bytes)
    local weth_hash
    weth_hash=$(python3 -c "
from hashlib import sha3_256
addr = bytes.fromhex('C02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2')
print('0x' + sha3_256(addr).hexdigest())
" 2>/dev/null || echo "")

    if [ -n "$weth_hash" ]; then
        echo "WETH account hash: $weth_hash"
        # Note: dumptrie requires state-root and storage-root which need the account data.
        # For now, just note the hash for manual inspection.
        echo "(Use 'geth db dumptrie <state-root> $weth_hash <storage-root> \"\" 5' for manual inspection)"
    else
        echo "Python3 not available for hash computation. Skipping dumptrie."
    fi
    echo ""
}

# ─── Main ────────────────────────────────────────────────────────────

check_deps

MODE="${1:-rpc}"

case "$MODE" in
    --db-only)
        run_db_checks
        ;;
    --all)
        run_rpc_checks
        echo ""
        echo "Stopping geth for database inspection..."
        kill "$(pgrep -f 'geth.*partial-test')" 2>/dev/null || true
        sleep 3
        run_db_checks
        ;;
    *)
        run_rpc_checks
        echo ""
        echo "For database-level verification, run:"
        echo "  $0 --db-only    (after stopping geth)"
        echo "  $0 --all        (stops geth automatically)"
        ;;
esac

exit $FAIL
