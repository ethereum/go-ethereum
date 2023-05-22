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

	// ScrollFeeVaultAddress is the address of the L2TxFeeVault
	// predeploy
	// see scroll-tech/scroll/contracts/src/L2/predeploys/L2TxFeeVault.sol
	ScrollFeeVaultAddress = common.HexToAddress("0x5300000000000000000000000000000000000005")

	// L1GasPriceOracleAddress is the address of the L1GasPriceOracle
	// predeploy
	// see scroll-tech/scroll/contracts/src/L2/predeploys/L1GasPriceOracle.sol
	L1GasPriceOracleAddress = common.HexToAddress("0x5300000000000000000000000000000000000002")
	Precision               = new(big.Int).SetUint64(1e9)
	L1BaseFeeSlot           = common.BigToHash(big.NewInt(1))
	OverheadSlot            = common.BigToHash(big.NewInt(2))
	ScalarSlot              = common.BigToHash(big.NewInt(3))
)
