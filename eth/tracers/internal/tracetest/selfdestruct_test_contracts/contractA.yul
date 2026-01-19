object "ContractA" {
    code {
        datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
        return(0, datasize("Runtime"))
    }
    object "Runtime" {
        code {
            // If receiving funds (msg.value > 0), just accept them and return
            if gt(callvalue(), 0) {
                stop()
            }

            // Otherwise, selfdestruct to B (transfers balance immediately, then stops execution)
            let contractB := 0x00000000000000000000000000000000000000bb
            selfdestruct(contractB)
        }
    }
}
