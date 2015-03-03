package ezp

import (
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethutil"
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

func (pow *EasyPow) Search(block pow.Block, stop <-chan struct{}) ([]byte, []byte, []byte) {
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
			return nil, nil, nil
		default:
			i++

			elapsed := time.Now().UnixNano() - start
			hashes := ((float64(1e9) / float64(elapsed)) * float64(i-starti)) / 1000
			pow.HashRate = int64(hashes)

			sha := crypto.Sha3(big.NewInt(r.Int63()).Bytes())
			if verify(hash, diff, sha) {
				return sha, nil, nil
			}
		}

		if !pow.turbo {
			time.Sleep(20 * time.Microsecond)
		}
	}
}

func (pow *EasyPow) Verify(block pow.Block) bool {
	return Verify(block)
}

func verify(hash []byte, diff *big.Int, nonce []byte) bool {
	sha := sha3.NewKeccak256()

	d := append(hash, nonce...)
	sha.Write(d)

	verification := new(big.Int).Div(ethutil.BigPow(2, 256), diff)
	res := ethutil.BigD(sha.Sum(nil))

	return res.Cmp(verification) <= 0
}

func Verify(block pow.Block) bool {
	return verify(block.HashNoNonce(), block.Difficulty(), block.Nonce())
}
