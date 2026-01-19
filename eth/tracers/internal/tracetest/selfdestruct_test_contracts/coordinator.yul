object "Coordinator" {
    code {
        datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
        return(0, datasize("Runtime"))
    }
    object "Runtime" {
        code {
            let contractA := 0x00000000000000000000000000000000000000aa
            let contractB := 0x00000000000000000000000000000000000000bb

            // First, call A (A will selfdestruct to B)
            pop(call(gas(), contractA, 0, 0, 0, 0, 0))

            // Then, call B (B will send funds back to A)
            pop(call(gas(), contractB, 0, 0, 0, 0, 0))

            stop()
        }
    }
}
