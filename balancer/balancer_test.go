package balancer

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func makeTxs(size int, b *testing.B) []types.Transactions {
	key, _ := crypto.GenerateKey()

	batches := make([]types.Transactions, b.N)
	for i := 0; i < b.N; i++ {
		txs := make(types.Transactions, size)
		for j := range txs {
			var err error
			txs[j], err = types.NewTransaction(0, common.Address{}, new(big.Int), new(big.Int), new(big.Int), nil).SignECDSA(key)
			if err != nil {
				b.Fatal(err)
			}
		}
		batches[i] = txs
	}
	return batches
}

const benchTxSize = 1024

func BenchmarkTxsRaw(b *testing.B) {
	batches := makeTxs(benchTxSize, b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		txs := batches[i]
		b.StartTimer()

		for _, tx := range txs {
			if _, err := tx.From(); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkTxsLoadBalancer(b *testing.B) {
	balancer := New(4)
	batches := makeTxs(benchTxSize, b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		txs := batches[i]

		var (
			size      = 8
			batchsize = len(txs) / size
			ch        = make(chan error, size)
		)

		for j := 0; j < size; j++ {
			j := j
			task := Task{
				fn: func() error {
					for _, tx := range txs[j*size : j*size+batchsize] {
						if _, err := tx.From(); err != nil {
							return err
						}
					}
					return nil
				},
				c: ch,
			}

			balancer.Push(task)
		}

		for j := 0; j < size; j++ {
			if err := <-ch; err != nil {
				b.Error(err)
			}
		}
		close(ch)
	}
}
