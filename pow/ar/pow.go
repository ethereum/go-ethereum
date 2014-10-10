package ar

import (
	"math/big"

	"github.com/ethereum/eth-go/ethutil"
)

type Entry struct {
	op   OpsFunc
	i, j *big.Int
}

type Tape struct {
	tape  []Entry
	block Block
}

func NewTape(block Block) *Tape {
	return &Tape{nil, block}
}

func (self *Tape) gen(w, h int64, gen NumberGenerator) {
	self.tape = nil

	for v := int64(0); v < h; v++ {
		op := ops[gen.rand64(lenops).Int64()]
		r := gen.rand64(100).Uint64()

		var j *big.Int
		if r < 20 && v > 20 {
			j = self.tape[len(self.tape)-1].i
		} else {
			j = gen.rand64(w)
		}

		i := gen.rand64(w)
		self.tape = append(self.tape, Entry{op, i, j})
	}
}

func (self *Tape) runTape(w, h int64, gen NumberGenerator) *big.Int {
	var mem []*big.Int
	for i := int64(0); i < w; i++ {
		mem = append(mem, gen.rand(ethutil.BigPow(2, 64)))
	}

	set := func(i, j int) Entry {
		entry := self.tape[i*100+j]
		mem[entry.i.Uint64()] = entry.op(entry.i, entry.j)

		return entry
	}

	dir := true
	for i := 0; i < int(h)/100; i++ {
		var entry Entry
		if dir {
			for j := 0; j < 100; j++ {
				entry = set(i, j)
			}
		} else {
			for j := 99; i >= 0; j-- {
				entry = set(i, j)
			}
		}

		t := mem[entry.i.Uint64()]
		if big.NewInt(2).Cmp(new(big.Int).Mod(t, big.NewInt(37))) < 0 {
			dir = !dir
		}
	}

	return Sha3(mem)
}

func (self *Tape) Verify(header, nonce []byte) bool {
	n := ethutil.BigD(nonce)

	var w int64 = 10000
	var h int64 = 150000
	gen := Rnd(Sha3([]interface{}{header, new(big.Int).Div(n, big.NewInt(1000))}))
	self.gen(w, h, gen)

	gen = Rnd(Sha3([]interface{}{header, new(big.Int).Mod(n, big.NewInt(1000))}))
	hash := self.runTape(w, h, gen)

	it := self.block.Trie().Iterator()
	next := it.Next(string(new(big.Int).Mod(hash, ethutil.BigPow(2, 160)).Bytes()))

	req := ethutil.BigPow(2, 256)
	req.Div(req, self.block.Diff())
	return Sha3([]interface{}{hash, next}).Cmp(req) < 0
}

func (self *Tape) Run(header []byte) []byte {
	nonce := big.NewInt(0)
	var w int64 = 10000
	var h int64 = 150000

	req := ethutil.BigPow(2, 256)
	req.Div(req, self.block.Diff())

	for {
		if new(big.Int).Mod(nonce, b(1000)).Cmp(b(0)) == 0 {
			gen := Rnd(Sha3([]interface{}{header, new(big.Int).Div(nonce, big.NewInt(1000))}))
			self.gen(w, h, gen)
		}

		gen := Rnd(Sha3([]interface{}{header, new(big.Int).Mod(nonce, big.NewInt(1000))}))
		hash := self.runTape(w, h, gen)

		it := self.block.Trie().Iterator()
		next := it.Next(string(new(big.Int).Mod(hash, ethutil.BigPow(2, 160)).Bytes()))

		if Sha3([]interface{}{hash, next}).Cmp(req) < 0 {
			return nonce.Bytes()
		} else {
			nonce.Add(nonce, ethutil.Big1)
		}
	}
}
