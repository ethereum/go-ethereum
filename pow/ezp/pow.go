package ezp

import (
	"encoding/binary"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
)

var powlogger = logger.NewLogger("POW")

type EasyPow struct {
	hash     *big.Int
	HashRate int64
	turbo    bool
}

func New() *EasyPow {
	return &EasyPow{turbo: false}
}

func (pow *EasyPow) GetHashrate() int64 {
	return pow.HashRate
}

func (pow *EasyPow) Turbo(on bool) {
	pow.turbo = on
}

func (pow *EasyPow) Search(block pow.Block, stop <-chan struct{}) (uint64, []byte) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := block.HashNoNonce()
	diff := block.Difficulty()
	//i := int64(0)
	// TODO fix offset
	i := rand.Int63()
	starti := i
	start := time.Now().UnixNano()

	defer func() { pow.HashRate = 0 }()

	// Make sure stop is empty
empty:
	for {
		select {
		case <-stop:
		default:
			break empty
		}
	}

	for {
		select {
		case <-stop:
			return 0, nil
		default:
			i++

			elapsed := time.Now().UnixNano() - start
			hashes := ((float64(1e9) / float64(elapsed)) * float64(i-starti)) / 1000
			pow.HashRate = int64(hashes)

			sha := uint64(r.Int63())
			if verify(hash, diff, sha) {
				return sha, nil
			}
		}

		if !pow.turbo {
			time.Sleep(20 * time.Microsecond)
		}
	}

	return 0, nil
}

func (pow *EasyPow) Verify(block pow.Block) bool {
	return Verify(block)
}

func verify(hash common.Hash, diff *big.Int, nonce uint64) bool {
	sha := sha3.NewKeccak256()
	n := make([]byte, 8)
	binary.PutUvarint(n, nonce)
	sha.Write(n)
	sha.Write(hash[:])
	verification := new(big.Int).Div(common.BigPow(2, 256), diff)
	res := common.BigD(sha.Sum(nil))
	return res.Cmp(verification) <= 0
}

func Verify(block pow.Block) bool {
	return verify(block.HashNoNonce(), block.Difficulty(), block.Nonce())
}
