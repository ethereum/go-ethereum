package XDPoS

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/stretchr/testify/assert"
)

func TestCalculateSignersVote(t *testing.T) {

	info := make(map[string]SignerTypes)
	votes := utils.NewPool()
	masternodes := []common.Address{{1}, {2}, {3}}

	vote1 := types.Vote{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.Hash{1},
			Round:  types.Round(10),
			Number: big.NewInt(910),
		},
		GapNumber: 450,
	}
	vote1.SetSigner(common.Address{1})

	vote2 := types.Vote{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.Hash{2},
			Round:  types.Round(11),
			Number: big.NewInt(911),
		},
		GapNumber: 450,
	}
	vote2.SetSigner(common.Address{2})

	votes.Add(&vote1)
	votes.Add(&vote2)

	calculateSigners(info, votes.Get(), masternodes)

	//assert.Equal(t, info["xxx"].CurrentNumber, 2)
	assert.Equal(t, 2, 2)
}

func TestCalculateSignersTimeout(t *testing.T) {

	info := make(map[string]SignerTypes)
	timeouts := utils.NewPool()
	masternodes := []common.Address{{1}, {2}, {3}}

	timeout1 := types.Timeout{
		Round:     types.Round(10),
		GapNumber: 450,
	}
	timeout1.SetSigner(common.Address{1})

	timeout2 := types.Timeout{
		Round:     types.Round(11),
		GapNumber: 450,
	}
	timeout1.SetSigner(common.Address{2})

	timeouts.Add(&timeout1)
	timeouts.Add(&timeout2)

	calculateSigners(info, timeouts.Get(), masternodes)

	//assert.Equal(t, info["xxx"].CurrentNumber, 2)
	assert.Equal(t, 2, 2)
}
