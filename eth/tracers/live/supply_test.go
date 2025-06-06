package live

import (
	"log"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/natefinch/lumberjack.v2"
)

func setup(t *testing.T) *supplyTracer {
	return &supplyTracer{
		chainConfig: &params.ChainConfig{
			LondonBlock: big.NewInt(0),
			PragueTime:  new(uint64),
			BlobScheduleConfig: &params.BlobScheduleConfig{
				Prague: params.DefaultPragueBlobConfig,
			},
		},
		delta: newSupplyInfo(),
		logger: &lumberjack.Logger{
			Filename: filepath.Join(t.TempDir(), "supply.jsonl"),
		},
	}
}

func TestGenesis(t *testing.T) {
	tracer := setup(t)
	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002")
	block := types.NewBlock(&types.Header{}, nil, nil, nil)
	alloc := types.GenesisAlloc{
		addr1: types.Account{Balance: big.NewInt(100)},
		addr2: types.Account{Balance: big.NewInt(200)},
	}

	tracer.onGenesisBlock(block, alloc)
	if tracer.delta.Issuance.GenesisAlloc.Cmp(big.NewInt(300)) != 0 {
		t.Fatal("Genesis block accounting error")
	}
}

func TestBalanceChanges(t *testing.T) {
	tracer := setup(t)
	addr := common.Address{}
	delta := big.NewInt(100)
	rewardTotal := big.NewInt(0)

	// block reward
	rewardTotal.Add(rewardTotal, delta)
	tracer.onBalanceChange(addr, big.NewInt(0), delta, tracing.BalanceIncreaseRewardMineBlock)
	if tracer.delta.Issuance.Reward.Cmp(rewardTotal) != 0 {
		log.Fatal("Block reward accounting error")
	}

	// uncle reward
	rewardTotal.Add(rewardTotal, delta)
	tracer.onBalanceChange(addr, big.NewInt(0), delta, tracing.BalanceIncreaseRewardMineUncle)
	if tracer.delta.Issuance.Reward.Cmp(rewardTotal) != 0 {
		log.Fatal("Uncle reward accounting error")
	}

	// withdrawal
	tracer.onBalanceChange(addr, big.NewInt(0), delta, tracing.BalanceIncreaseWithdrawal)
	if tracer.delta.Issuance.Withdrawals.Cmp(delta) != 0 {
		log.Fatal("Withdrawal accounting error")
	}

	// self destruct burn
	tracer.onBalanceChange(addr, delta, big.NewInt(0), tracing.BalanceDecreaseSelfdestructBurn)
	if tracer.delta.Burn.Misc.Cmp(delta) != 0 {
		log.Fatal("Self destruct burn accounting error")
	}
}

func TestBlockBurns(t *testing.T) {
	tracer := setup(t)
	blobGasUsed := uint64(10)
	excessBlobGas := uint64(90)
	block := types.NewBlock(&types.Header{
		BaseFee: big.NewInt(100),
		GasUsed: 10,

		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
		Time:          1,
	}, nil, nil, nil)

	tracer.onBlockStart(tracing.BlockEvent{Block: block})

	// eip1559 burn
	if tracer.delta.Burn.EIP1559.Cmp(big.NewInt(1000)) != 0 {
		log.Fatal("EIP1559 burn accounting error")
	}

	// blob burn
	blobFee := eip4844.CalcBlobFee(tracer.chainConfig, block.Header())
	blobBurn := new(big.Int).Mul(new(big.Int).SetUint64(blobGasUsed), blobFee)
	if tracer.delta.Burn.Blob.Cmp(blobBurn) != 0 {
		log.Fatal("Blob burn accounting error")
	}
}

func TestTransactionBurns(t *testing.T) {
	tracer := setup(t)
	addr := common.Address{}
	runTx := func(reverted bool) {
		tracer.onTxStart(nil, nil, addr)
		tracer.onEnter(0, byte(vm.SELFDESTRUCT), addr, addr, nil, 0, big.NewInt(100))
		tracer.onExit(0, nil, 0, nil, reverted)
	}

	runTx(false)
	runTx(false)
	runTx(true) // reverted transactions not accounted

	if tracer.delta.Burn.Misc.Cmp(big.NewInt(200)) != 0 {
		log.Fatal("Self destruct accounting error")
	}
}
