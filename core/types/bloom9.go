package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
)

type bytesBacked interface {
	Bytes() []byte
}

func CreateBloom(receipts Receipts) Bloom {
	bin := new(big.Int)
	for _, receipt := range receipts {
		bin.Or(bin, LogsBloom(receipt.logs))
	}

	return BytesToBloom(bin.Bytes())
}

func LogsBloom(logs state.Logs) *big.Int {
	bin := new(big.Int)
	for _, log := range logs {
		data := make([]common.Hash, len(log.Topics))
		bin.Or(bin, bloom9(log.Address.Bytes()))

		for i, topic := range log.Topics {
			data[i] = topic
		}

		for _, b := range data {
			bin.Or(bin, bloom9(b[:]))
		}
	}

	return bin
}

func bloom9(b []byte) *big.Int {
	b = crypto.Sha3(b[:])

	r := new(big.Int)

	for i := 0; i < 6; i += 2 {
		t := big.NewInt(1)
		b := (uint(b[i+1]) + (uint(b[i]) << 8)) & 2047
		r.Or(r, t.Lsh(t, b))
	}

	return r
}

var Bloom9 = bloom9

func BloomLookup(bin Bloom, topic bytesBacked) bool {
	bloom := bin.Big()
	cmp := bloom9(topic.Bytes()[:])

	return bloom.And(bloom, cmp).Cmp(cmp) == 0
}
