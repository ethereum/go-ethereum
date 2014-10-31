package chain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/ethstate"
	"github.com/ethereum/go-ethereum/ethutil"
)

func CreateBloom(block *Block) []byte {
	bin := new(big.Int)
	bin.Or(bin, bloom9(block.Coinbase))
	for _, receipt := range block.Receipts() {
		bin.Or(bin, LogsBloom(receipt.logs))
	}

	return bin.Bytes()
}

func LogsBloom(logs ethstate.Logs) *big.Int {
	bin := new(big.Int)
	for _, log := range logs {
		data := [][]byte{log.Address}
		for _, topic := range log.Topics {
			data = append(data, topic)
		}

		if log.Data != nil {
			data = append(data, log.Data)
		}

		for _, b := range data {
			bin.Or(bin, bloom9(b))
		}
	}

	return bin
}

func bloom9(b []byte) *big.Int {
	r := new(big.Int)
	for _, i := range []int{0, 2, 4} {
		t := big.NewInt(1)
		r.Or(r, t.Lsh(t, uint(b[i+1])+256*(uint(b[i])&1)))
	}

	return r
}

func BloomLookup(bin, topic []byte) bool {
	bloom := ethutil.BigD(bin)
	cmp := bloom9(topic)

	return bloom.And(bloom, cmp).Cmp(cmp) == 0
}
