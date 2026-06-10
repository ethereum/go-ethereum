#!/usr/bin/env bash
# Regenerates core/vm/runtime/testdata/*.hex, the runtime bytecode for the
# whole-contract interpreter benchmarks (see core/vm/runtime/evm_bench_test.go).
#
# Source: the evm-bench suite (github.com/ziyadedher/evm-bench). Each contract
# exposes Benchmark() (selector 0x30627b7c) which performs the whole workload.
# Compiled with solc via docker (no local solc needed). solc versions match each
# contract's pragma: 0.8.17 for the ERC-20/hashing contracts, 0.4.26 for the
# legacy SnailTracer.
#
# Usage: core/vm/runtime/testdata/evm-bench/gen.sh
set -euo pipefail

TD="$(cd "$(dirname "$0")/.." && pwd)"   # .../core/vm/runtime/testdata
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

git clone --depth 1 https://github.com/ziyadedher/evm-bench "$WORK/evm-bench" >/dev/null 2>&1
B="$WORK/evm-bench/benchmarks"

# ERC-20 (transfer/mint/approval) + ten-thousand-hashes: pragma ^0.8.17. These
# set up all their state inside Benchmark(), so the runtime bytecode is callable
# directly with no constructor, which is how the benchmark drives them.
docker run --rm -v "$B":/src -w /src ethereum/solc:0.8.17 \
	--optimize --bin-runtime --overwrite -o /src/out \
	erc20/transfer/ERC20Transfer.sol \
	erc20/mint/ERC20Mint.sol \
	erc20/approval-transfer/ERC20ApprovalTransfer.sol \
	ten-thousand-hashes/TenThousandHashes.sol

cp "$B/out/TenThousandHashes.bin-runtime"     "$TD/tenthousandhashes.hex"
cp "$B/out/ERC20Transfer.bin-runtime"         "$TD/erc20transfer.hex"
cp "$B/out/ERC20Mint.bin-runtime"             "$TD/erc20mint.hex"
cp "$B/out/ERC20ApprovalTransfer.bin-runtime" "$TD/erc20approval.hex"

# NOTE: snailtracer.hex is not regenerated here. evm-bench's SnailTracer
# initializes its scene in the constructor, so its --bin-runtime (no constructor)
# renders an empty scene. The committed snailtracer.hex is a runtime-callable
# build (scene encoded in code, Benchmark() self-contained) vendored from
# Giulio2002/gevm's gethbench testdata. Leave it as-is.

echo "regenerated (snailtracer.hex left vendored):"
ls -l "$TD"/*.hex
