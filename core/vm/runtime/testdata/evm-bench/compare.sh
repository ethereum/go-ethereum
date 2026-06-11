#!/usr/bin/env bash
# A/B benchmark of the codegen interpreter vs a baseline: runs the evm-bench
# contract workloads (core/vm/runtime/evm_bench_test.go) plus the block import
# benchmark (core/bench_test.go, BenchmarkInsertChain_evmWorkload, a local
# stand-in for sync throughput) on the current working tree and on a baseline
# ref, then benchstats them.
#
# The baseline is checked out in a throwaway git worktree and this suite is
# copied into it, so the comparison works whether or not the interpreter changes
# are committed yet. Requires: go, git, and benchstat
# (go install golang.org/x/perf/cmd/benchstat@latest).
#
# Usage: core/vm/runtime/testdata/evm-bench/compare.sh [baseref] [count]
#        baseref defaults to "master", count to 10.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../../.." && pwd)"   # repo root
cd "$ROOT"
BASEREF="${1:-master}"
COUNT="${2:-10}"
# Each benchmark runs a FIXED iteration count instead of the default 1s of
# benchtime. With time-based benchtime the faster side runs more iterations,
# so one-time costs (map growth, pool warmup) amortize over a different N and
# B/op picks up phantom deltas, and GC pacing can do the same to sec/op. Fixed
# N makes both sides do identical work. The counts target roughly one second
# per count on a fast box. Each entry is package:pattern:iterations.
BENCHES=(
	'./core/vm/runtime/:^BenchmarkSnailtracer$:20x'
	'./core/vm/runtime/:^BenchmarkTenThousandHashes$:100x'
	'./core/vm/runtime/:^BenchmarkERC20Transfer$:100x'
	'./core/vm/runtime/:^BenchmarkERC20Mint$:150x'
	'./core/vm/runtime/:^BenchmarkERC20ApprovalTransfer$:120x'
	'./core/:^BenchmarkInsertChain_evmWorkload_memdb$:10x'
	'./core/:^BenchmarkInsertChain_evmWorkload_diskdb$:10x'
)
NEW="$(mktemp)"; OLD="$(mktemp)"

run_suite() { # run_suite <dir> <outfile>
	local dir="$1" out="$2" entry pkg pat n
	for entry in "${BENCHES[@]}"; do
		pkg="${entry%%:*}"
		pat="${entry#*:}"; pat="${pat%:*}"
		n="${entry##*:}"
		( cd "$dir" && go test "$pkg" -run '^$' -bench "$pat" -benchmem -benchtime "$n" -count="$COUNT" ) | tee -a "$out"
	done
}

echo "==> current working tree"
run_suite "$ROOT" "$NEW"

echo "==> baseline: $BASEREF (throwaway worktree, suite copied in)"
WT="$(mktemp -d)"
git worktree add --quiet --detach "$WT" "$BASEREF"
cp core/vm/runtime/evm_bench_test.go "$WT/core/vm/runtime/"
cp core/bench_test.go "$WT/core/bench_test.go"
mkdir -p "$WT/core/vm/runtime/testdata"
cp core/vm/runtime/testdata/*.hex "$WT/core/vm/runtime/testdata/"
run_suite "$WT" "$OLD"
git worktree remove --force "$WT"

echo "==> benchstat: $BASEREF (left) vs working tree (right)"
benchstat "$OLD" "$NEW"
