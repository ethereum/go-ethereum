package posv

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestGetM1M2FromCheckpointHeader(t *testing.T) {
	masternodes := []common.Address{
		common.StringToAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
	}
	validators := []int64{
		2,
		1,
		0,
	}
	epoch := uint64(900)
	config := &params.ChainConfig{
		Posv: &params.PosvConfig{
			Epoch: uint64(epoch),
		},
	}
	//try from block 900 to 909
	for i := uint64(3410001); i < 3411027; i++ {
		currentNumber := int64(i)
		currentHeader := &types.Header{
			Number: big.NewInt(currentNumber),
		}
		m1m2, moveM2, err := getM1M2(masternodes, validators, currentHeader, config)
		if err != nil {
			t.Error("can't get m1m2", "err", err)
		}
		fmt.Printf("block: %v, moveM2: %v\n", currentHeader.Number.Int64(), moveM2)
		for _, k := range masternodes {
			fmt.Printf("m1: %v - m2: %v\n", k.Str(), m1m2[k].Str())
		}
		maxMNs := len(masternodes)
		testMoveM2 := ((uint64(currentNumber) % config.Posv.Epoch) / uint64(maxMNs)) % uint64(maxMNs)
		if moveM2 != testMoveM2 { //3 = len(masternodes)
			t.Error("wrong moveM2", "currentNumber", currentNumber, "want", testMoveM2, "have", moveM2)
		}
	}
}
