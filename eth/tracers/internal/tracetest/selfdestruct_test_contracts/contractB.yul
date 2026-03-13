object "ContractB" {
    code {
        datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
        return(0, datasize("Runtime"))
    }
    object "Runtime" {
        code {
            // Send 50 wei back to contract A
            let contractA := 0x00000000000000000000000000000000000000aa
            let success := call(gas(), contractA, 50, 0, 0, 0, 0)
            stop()
        }
    }
}
