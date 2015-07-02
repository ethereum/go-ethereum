package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	jeff      = common.HexToAddress("a8edb1ac2c86d3d9d78f96cd18001f60df29e52c")
	vitalik   = common.HexToAddress("1baf27b88c48dd02b744999cf3522766929d2b2a")
	christoph = common.HexToAddress("60d11b58744784dc97f878f7e3749c0f1381a004")
	gav       = common.HexToAddress("4bb7e8ae99b645c2b7860b8f3a2328aae28bd80a")
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
