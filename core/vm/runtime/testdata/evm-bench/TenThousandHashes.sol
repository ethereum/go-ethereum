// SPDX-License-Identifier: GPL-3.0
pragma solidity ^0.8.17;

// Vendored from evm-bench (github.com/ziyadedher/evm-bench) with one fix.
// The upstream loop discards the keccak256 result, so solc's optimizer
// removes the pure call entirely and the compiled benchmark degenerates
// into a bare counter loop that performs a single static hash. Chaining
// the hash through an accumulator that the function returns keeps all
// 20000 hashes in the optimized bytecode. The Benchmark() selector is
// unchanged because return types are not part of the signature.
contract TenThousandHashes {
    function Benchmark() external pure returns (bytes32 acc) {
        for (uint256 i = 0; i < 20000; i++) {
            acc = keccak256(abi.encodePacked(acc, i));
        }
    }
}
