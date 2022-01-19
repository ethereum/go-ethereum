package server

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
	"github.com/stretchr/testify/assert"
)

func TestGatherBlocks(t *testing.T) {
	type c struct {
		ABlock *big.Int
		BBlock *big.Int
	}
	type d struct {
		DBlock uint64
	}
	val := &c{
		BBlock: new(big.Int).SetInt64(1),
	}
	val2 := &d{
		DBlock: 10,
	}

	expect := []*proto.StatusResponse_Fork{
		{
			Name:     "A",
			Disabled: true,
		},
		{
			Name:  "B",
			Block: 1,
		},
		{
			Name:  "D",
			Block: 10,
		},
	}

	res := gatherForks(val, val2)
	assert.Equal(t, res, expect)
}
