package chain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

func CreateBloom(block *Block) []byte {
	bin := new(big.Int)
	for _, receipt := range block.Receipts() {
		bin.Or(bin, LogsBloom(receipt.logs))
	}

	return ethutil.LeftPadBytes(bin.Bytes(), 64)
}

func LogsBloom(logs state.Logs) *big.Int {
	bin := new(big.Int)
	for _, log := range logs {
		data := [][]byte{log.Address}
		for _, topic := range log.Topics {
			data = append(data, topic)
		}

		for _, b := range data {
			bin.Or(bin, ethutil.BigD(bloom9(crypto.Sha3(b)).Bytes()))
		}

		//if log.Data != nil {
		//	data = append(data, log.Data)
		//}
	}

	return bin
}

func bloom9(b []byte) *big.Int {
	r := new(big.Int)
	for _, i := range []int{0, 2, 4} {
		t := big.NewInt(1)
		b := uint(b[i+1]) + 256*(uint(b[i])&1)
		r.Or(r, t.Lsh(t, b))
	}

	return r
}

func BloomLookup(bin, topic []byte) bool {
	bloom := ethutil.BigD(bin)
	cmp := bloom9(crypto.Sha3(topic))

	return bloom.And(bloom, cmp).Cmp(cmp) == 0
}
