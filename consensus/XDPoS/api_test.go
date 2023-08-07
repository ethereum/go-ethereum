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
		Signature: types.Signature{1},
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.Hash{1},
			Round:  types.Round(10),
			Number: big.NewInt(910),
		},
		GapNumber: 450,
	}
	vote1.SetSigner(common.Address{1})

	vote2 := types.Vote{
		Signature: types.Signature{2},
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.Hash{1},
			Round:  types.Round(10),
			Number: big.NewInt(910),
		},
		GapNumber: 450,
	}
	vote2.SetSigner(common.Address{2})

	votes.Add(&vote1)
	votes.Add(&vote2)

	calculateSigners(info, votes.Get(), masternodes)
	assert.Equal(t, info["10:450:910:0x0100000000000000000000000000000000000000000000000000000000000000"].CurrentNumber, 2)
}

func TestCalculateSignersTimeout(t *testing.T) {

	info := make(map[string]SignerTypes)
	timeouts := utils.NewPool()
	masternodes := []common.Address{{1}, {2}, {3}}

	timeout1 := types.Timeout{
		Signature: types.Signature{1},
		Round:     types.Round(10),
		GapNumber: 450,
	}
	timeout1.SetSigner(common.Address{1})

	timeout2 := types.Timeout{
		Signature: types.Signature{2},
		Round:     types.Round(10),
		GapNumber: 450,
	}
	timeout1.SetSigner(common.Address{2})

	timeouts.Add(&timeout1)
	timeouts.Add(&timeout2)

	calculateSigners(info, timeouts.Get(), masternodes)
	assert.Equal(t, info["10:450"].CurrentNumber, 2)
}
