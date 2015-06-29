package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	jeff      = common.HexToAddress("9d38997c624a71b21278389ea2fdc460d000e4b2")
	vitalik   = common.HexToAddress("b1e570be07eaa673e4fd0c8265b64ef739385709")
	christoph = common.HexToAddress("529bc43a5d93789fa28de1961db6a07e752204ae")
	gav       = common.HexToAddress("e3e942b2aa524293c84ff6c7f87a6635790ad5e4")
)

// Canary will check the 0'd address of the 4 contracts above.
// If two or more are set to anything other than a 0 the canary
// dies a horrible death.
func Canary(statedb *state.StateDB) bool {
	r := new(big.Int)
	r.Add(r, statedb.GetState(jeff, common.Hash{}).Big())
	r.Add(r, statedb.GetState(vitalik, common.Hash{}).Big())
	r.Add(r, statedb.GetState(christoph, common.Hash{}).Big())
	r.Add(r, statedb.GetState(gav, common.Hash{}).Big())

	return r.Cmp(big.NewInt(1)) > 0
}
