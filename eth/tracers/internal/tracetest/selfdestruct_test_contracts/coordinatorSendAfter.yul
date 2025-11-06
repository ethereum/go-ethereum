object "CoordinatorSendAfter" {
    code {
        datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
        return(0, datasize("Runtime"))
    }
    object "Runtime" {
        code {
            let contractAddr := 0x00000000000000000000000000000000000000aa

            // Call contract (triggers selfdestruct to self, burning its balance)
            pop(call(gas(), contractAddr, 0, 0, 0, 0, 0))

            // Check contract's balance immediately after selfdestruct
            // Store in slot 0 to verify it's 0 (proving immediate burn)
            sstore(0, balance(contractAddr))

            // Send 50 wei to the contract (after it selfdestructed)
            pop(call(gas(), contractAddr, 50, 0, 0, 0, 0))

            // Check balance again after sending funds
            // Store in slot 1 to verify it's 50 (new funds not burnt)
            sstore(1, balance(contractAddr))

            stop()
        }
    }
}
