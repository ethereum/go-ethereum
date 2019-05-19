package bmt

import (
	"bytes"
	"fmt"
	bmt2 "github.com/ethereum/go-ethereum/swarm/bmt"
	"golang.org/x/crypto/sha3"
)

func f1(pool *bmt2.TreePool, input []byte) (int, []byte) {
	b := bmt2.New(pool)
	b.Reset()
	_, err := b.Write(input)
	if err != nil {
		return 0, nil
	}
	x := make([]byte, 0, 512)
	_, err = b.Write(x)
	if err != nil {
		return 1, nil
	}
	return 2, b.Sum(nil)
}

func f2(pool *bmt2.TreePool, input []byte) (int, []byte) {
	b := bmt2.New(pool)
	b.Reset()
	for _, val := range input {
		_, err := b.Write([]byte{val})
		if err != nil {
			return 0, nil
		}
	}
	x := make([]byte, 0, 512)
	_, err := b.Write(x)
	if err != nil {
		return 1, nil
	}
	return 2, b.Sum(nil)
}

func Fuzz(input []byte) int {
	hasher := sha3.NewLegacyKeccak256
	pool := bmt2.NewTreePool(hasher, 128, bmt2.PoolSize)
	input2 := make([]byte, len(input))
	copy(input2, input)

	ret1, sum1 := f1(pool, input)

	ret2, sum2 := f2(pool, input)

	if ret1 != ret2 {
		panic(fmt.Sprintf("ret1: %d !=  ret2: %d", ret1, ret2))
	}
	if ret1 == 2 && !bytes.Equal(sum1, sum2) {
		panic("sums does not match")
	}
	return 0
}
