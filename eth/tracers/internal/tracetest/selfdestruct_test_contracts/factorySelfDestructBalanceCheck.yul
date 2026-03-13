object "FactorySelfDestructBalanceCheck" {
    code {
        datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
        return(0, datasize("Runtime"))
    }
    object "Runtime" {
        code {
            // Get the full deploy bytecode for ContractSelfDestruct
            // Compiled with: solc --strict-assembly --evm-version paris contractSelfDestruct.yul --bin
            // Full bytecode: 6002600d60003960026000f3fe30ff
            // That's 15 bytes total, padded to 32 bytes with 17 zero bytes at front
            mstore(0, 0x0000000000000000000000000000000000000000006002600d60003960026000f3fe30ff)

            // CREATE contract with 100 wei, using deploy bytecode
            // The bytecode is 15 bytes, starts at position 17 in the 32-byte word
            let contractAddr := create(100, 17, 15)

            // Call the created contract (triggers selfdestruct to self)
            pop(call(gas(), contractAddr, 0, 0, 0, 0, 0))

            // Check contract's balance immediately after selfdestruct
            // Store in slot 0 to verify it's 0 (proving immediate burn)
            sstore(0, balance(contractAddr))

            // Send 50 wei to the contract (after it selfdestructed)
            pop(call(gas(), contractAddr, 50, 0, 0, 0, 0))

            // Check balance again after sending funds
            // Store in slot 1 to verify it's 0 (funds sent to destroyed contract are burnt)
            sstore(1, balance(contractAddr))

            stop()
        }
    }
}
