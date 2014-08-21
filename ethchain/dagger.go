package ethchain

import (
	"hash"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethreact"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/sha3"
)

var powlogger = ethlog.NewLogger("POW")

type PoW interface {
	Search(block *Block, reactChan chan ethreact.Event) []byte
	Verify(hash []byte, diff *big.Int, nonce []byte) bool
	GetHashrate() int64
	Turbo(bool)
}

type EasyPow struct {
	hash     *big.Int
	HashRate int64
	turbo    bool
}

func (pow *EasyPow) GetHashrate() int64 {
	return pow.HashRate
}

func (pow *EasyPow) Turbo(on bool) {
	pow.turbo = on
}

func (pow *EasyPow) Search(block *Block, reactChan chan ethreact.Event) []byte {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := block.HashNoNonce()
	diff := block.Difficulty
	i := int64(0)
	start := time.Now().UnixNano()
	t := time.Now()

	for {
		select {
		case <-reactChan:
			powlogger.Infoln("Breaking from mining")
			return nil
		default:
			i++

			if time.Since(t) > (1 * time.Second) {
				elapsed := time.Now().UnixNano() - start
				hashes := ((float64(1e9) / float64(elapsed)) * float64(i)) / 1000
				pow.HashRate = int64(hashes)
				powlogger.Infoln("Hashing @", int64(pow.HashRate), "khash")

				t = time.Now()
			}

			sha := ethcrypto.Sha3Bin(big.NewInt(r.Int63()).Bytes())
			if pow.Verify(hash, diff, sha) {
				return sha
			}
		}

		if !pow.turbo {
			time.Sleep(20 * time.Microsecond)
		}
	}

	return nil
}

func (pow *EasyPow) Verify(hash []byte, diff *big.Int, nonce []byte) bool {
	sha := sha3.NewKeccak256()

	d := append(hash, nonce...)
	sha.Write(d)

	v := ethutil.BigPow(2, 256)
	ret := new(big.Int).Div(v, diff)

	res := new(big.Int)
	res.SetBytes(sha.Sum(nil))

	return res.Cmp(ret) == -1
}

func (pow *EasyPow) SetHash(hash *big.Int) {
}

type Dagger struct {
	hash *big.Int
	xn   *big.Int
}

var Found bool

func (dag *Dagger) Find(obj *big.Int, resChan chan int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 1000; i++ {
		rnd := r.Int63()

		res := dag.Eval(big.NewInt(rnd))
		powlogger.Infof("rnd %v\nres %v\nobj %v\n", rnd, res, obj)
		if res.Cmp(obj) < 0 {
			// Post back result on the channel
			resChan <- rnd
			// Notify other threads we've found a valid nonce
			Found = true
		}

		// Break out if found
		if Found {
			break
		}
	}

	resChan <- 0
}

func (dag *Dagger) Search(hash, diff *big.Int) *big.Int {
	// TODO fix multi threading. Somehow it results in the wrong nonce
	amountOfRoutines := 1

	dag.hash = hash

	obj := ethutil.BigPow(2, 256)
	obj = obj.Div(obj, diff)

	Found = false
	resChan := make(chan int64, 3)
	var res int64

	for k := 0; k < amountOfRoutines; k++ {
		go dag.Find(obj, resChan)

		// Wait for each go routine to finish
	}
	for k := 0; k < amountOfRoutines; k++ {
		// Get the result from the channel. 0 = quit
		if r := <-resChan; r != 0 {
			res = r
		}
	}

	return big.NewInt(res)
}

func (dag *Dagger) Verify(hash, diff, nonce *big.Int) bool {
	dag.hash = hash

	obj := ethutil.BigPow(2, 256)
	obj = obj.Div(obj, diff)

	return dag.Eval(nonce).Cmp(obj) < 0
}

func DaggerVerify(hash, diff, nonce *big.Int) bool {
	dagger := &Dagger{}
	dagger.hash = hash

	obj := ethutil.BigPow(2, 256)
	obj = obj.Div(obj, diff)

	return dagger.Eval(nonce).Cmp(obj) < 0
}

func (dag *Dagger) Node(L uint64, i uint64) *big.Int {
	if L == i {
		return dag.hash
	}

	var m *big.Int
	if L == 9 {
		m = big.NewInt(16)
	} else {
		m = big.NewInt(3)
	}

	sha := sha3.NewKeccak256()
	sha.Reset()
	d := sha3.NewKeccak256()
	b := new(big.Int)
	ret := new(big.Int)

	for k := 0; k < int(m.Uint64()); k++ {
		d.Reset()
		d.Write(dag.hash.Bytes())
		d.Write(dag.xn.Bytes())
		d.Write(big.NewInt(int64(L)).Bytes())
		d.Write(big.NewInt(int64(i)).Bytes())
		d.Write(big.NewInt(int64(k)).Bytes())

		b.SetBytes(Sum(d))
		pk := b.Uint64() & ((1 << ((L - 1) * 3)) - 1)
		sha.Write(dag.Node(L-1, pk).Bytes())
	}

	ret.SetBytes(Sum(sha))

	return ret
}

func Sum(sha hash.Hash) []byte {
	//in := make([]byte, 32)
	return sha.Sum(nil)
}

func (dag *Dagger) Eval(N *big.Int) *big.Int {
	pow := ethutil.BigPow(2, 26)
	dag.xn = pow.Div(N, pow)

	sha := sha3.NewKeccak256()
	sha.Reset()
	ret := new(big.Int)

	for k := 0; k < 4; k++ {
		d := sha3.NewKeccak256()
		b := new(big.Int)

		d.Reset()
		d.Write(dag.hash.Bytes())
		d.Write(dag.xn.Bytes())
		d.Write(N.Bytes())
		d.Write(big.NewInt(int64(k)).Bytes())

		b.SetBytes(Sum(d))
		pk := (b.Uint64() & 0x1ffffff)

		sha.Write(dag.Node(9, pk).Bytes())
	}

	return ret.SetBytes(Sum(sha))
}
