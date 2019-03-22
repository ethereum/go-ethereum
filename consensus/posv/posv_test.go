package posv

import (
	"testing"
	"math/big"
	"fmt"

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
	epoch := int64(900)
	config := &params.ChainConfig{
		Posv: &params.PosvConfig{
			Epoch: uint64(epoch),
		},
	}
	//try from block 900 to 909
	for i:=int64(0); i<9; i++ {
		currentHeader := &types.Header{
			Number: big.NewInt(epoch+i),
		}
		m1m2, moveM2, err := getM1M2(masternodes, validators, currentHeader, config)
		if err != nil {
			t.Error("can't get m1m2", "err", err)
		}
		fmt.Printf("block: %v, moveM2: %v\n", currentHeader.Number.Int64(), moveM2)
		for _,k := range masternodes {
			fmt.Printf("m1: %v - m2: %v\n", k.Str(), m1m2[k].Str())
		}
		if moveM2 != uint64(i/3) { //3 = len(masternodes)
			t.Error("wrong moveM2", "want", uint64(i/3), "have", moveM2)
		}
	}
}
