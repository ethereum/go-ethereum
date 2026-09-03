object "FactoryRefund" {
    code {
        datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
        return(0, datasize("Runtime"))
    }
    object "Runtime" {
        code {
            let contractB := 0x00000000000000000000000000000000000000bb

            // Store the deploy bytecode for contract A in memory
            // Full deploy bytecode from: solc --strict-assembly --evm-version paris contractA.yul --bin
            // Including the 0xfe separator: 600c600d600039600c6000f3fe60003411600a5760bbff5b00
            // That's 25 bytes, padded to 32 bytes with 7 zero bytes at the front
            mstore(0, 0x0000000000000000000000000000600c600d600039600c6000f3fe60003411600a5760bbff5b00)

            // CREATE contract A with 100 wei, using 25 bytes starting at position 7
            let contractA := create(100, 7, 25)

            // Call contract A (triggers selfdestruct to B)
            pop(call(gas(), contractA, 0, 0, 0, 0, 0))

            // Call contract B (B sends 50 wei back to A)
            pop(call(gas(), contractB, 0, 0, 0, 0, 0))

            stop()
        }
    }
}
