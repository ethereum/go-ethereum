object "ContractSelfDestruct" {
    code {
        datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
        return(0, datasize("Runtime"))
    }
    object "Runtime" {
        code {
            // Simply selfdestruct to self
            selfdestruct(address())
        }
    }
}
