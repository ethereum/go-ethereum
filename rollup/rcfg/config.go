package rcfg

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

// TODO:
// verify in consensus layer when decentralizing sequencer

var (
	// L2MessageQueueAddress is the address of the L2MessageQueue
	// predeploy
	// see contracts/src/L2/predeploys/L2MessageQueue.sol
	L2MessageQueueAddress = common.HexToAddress("0x5300000000000000000000000000000000000000")
	WithdrawTrieRootSlot  = common.BigToHash(big.NewInt(0))
)
