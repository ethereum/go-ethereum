package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/golang-lru"
)

var (
	segments            *lru.Cache
	DisableSegmentation bool
)

func init() {
	segments, _ = lru.New(256)
}

type codeSegments map[uint64]*segment

// operation is a "native" implementation of an OpCode
type operation struct {
	fn   func(*stack, []byte)
	data []byte
}

type segment struct {
	cstart, cend, next uint64
	msize              uint64
	gas                *big.Int

	ssize int

	ops []operation
}

func (c *segment) String() string {
	return fmt.Sprintf("{%d %d %d %v}", c.cstart, c.cend, c.msize, c.gas)
}

// native operations

func opPush(s *stack, data []byte) {
	s.push(common.BytesToBig(data))
}

func opAdd(s *stack, data []byte) {
	s.push(U256(new(big.Int).Add(s.pop(), s.pop())))
}

func opSub(s *stack, data []byte) {
	s.push(U256(new(big.Int).Sub(s.pop(), s.pop())))
}

func opMul(s *stack, data []byte) {
	s.push(U256(new(big.Int).Mul(s.pop(), s.pop())))
}

func opDiv(s *stack, data []byte) {
	x, y := s.pop(), s.pop()

	if y.Cmp(common.Big0) != 0 {
		new(big.Int).Div(x, y)
	}

	U256(new(big.Int))

	// pop result back on the s
	s.push(new(big.Int))
}

func opSdiv(s *stack, data []byte) {
	x, y := S256(s.pop()), S256(s.pop())

	if y.Cmp(common.Big0) == 0 {
		new(big.Int).Set(common.Big0)
	} else {
		n := new(big.Int)
		if new(big.Int).Mul(x, y).Cmp(common.Big0) < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		new(big.Int).Div(x.Abs(x), y.Abs(y)).Mul(new(big.Int), n)

		U256(new(big.Int))
	}

	s.push(new(big.Int))
}

func opMod(s *stack, data []byte) {
	x, y := s.pop(), s.pop()

	if y.Cmp(common.Big0) == 0 {
		new(big.Int).Set(common.Big0)
	} else {
		new(big.Int).Mod(x, y)
	}

	U256(new(big.Int))

	s.push(new(big.Int))
}

func opSmod(s *stack, data []byte) {
	x, y := S256(s.pop()), S256(s.pop())

	if y.Cmp(common.Big0) == 0 {
		new(big.Int).Set(common.Big0)
	} else {
		n := new(big.Int)
		if x.Cmp(common.Big0) < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		new(big.Int).Mod(x.Abs(x), y.Abs(y)).Mul(new(big.Int), n)

		U256(new(big.Int))
	}

	s.push(new(big.Int))

}

func opExp(s *stack, data []byte) {
	x, y := s.pop(), s.pop()

	new(big.Int).Exp(x, y, Pow256)

	U256(new(big.Int))

	s.push(new(big.Int))
}

func opSignextend(s *stack, data []byte) {
	back := s.pop()
	if back.Cmp(big.NewInt(31)) < 0 {
		bit := uint(back.Uint64()*8 + 7)
		num := s.pop()
		mask := new(big.Int).Lsh(common.Big1, bit)
		mask.Sub(mask, common.Big1)
		if common.BitTest(num, int(bit)) {
			num.Or(num, mask.Not(mask))
		} else {
			num.And(num, mask)
		}

		num = U256(num)

		s.push(num)
	}
}

func opNot(s *stack, data []byte) {
	s.push(U256(new(big.Int).Not(s.pop())))
}

func opLt(s *stack, data []byte) {
	x, y := s.pop(), s.pop()

	// x < y
	if x.Cmp(y) < 0 {
		s.push(common.BigTrue)
	} else {
		s.push(common.BigFalse)
	}
}

func opGt(s *stack, data []byte) {
	x, y := s.pop(), s.pop()

	// x > y
	if x.Cmp(y) > 0 {
		s.push(common.BigTrue)
	} else {
		s.push(common.BigFalse)
	}
}

func opSlt(s *stack, data []byte) {
	x, y := S256(s.pop()), S256(s.pop())

	// x < y
	if x.Cmp(S256(y)) < 0 {
		s.push(common.BigTrue)
	} else {
		s.push(common.BigFalse)
	}
}

func opSgt(s *stack, data []byte) {
	x, y := S256(s.pop()), S256(s.pop())

	// x > y
	if x.Cmp(y) > 0 {
		s.push(common.BigTrue)
	} else {
		s.push(common.BigFalse)
	}
}

func opEq(s *stack, data []byte) {
	x, y := s.pop(), s.pop()

	// x == y
	if x.Cmp(y) == 0 {
		s.push(common.BigTrue)
	} else {
		s.push(common.BigFalse)
	}
}

func opIszero(s *stack, data []byte) {
	x := s.pop()
	if x.Cmp(common.BigFalse) > 0 {
		s.push(common.BigFalse)
	} else {
		s.push(common.BigTrue)
	}
}

func opAnd(s *stack, data []byte) {
	s.push(new(big.Int).And(s.pop(), s.pop()))
}

func opOr(s *stack, data []byte) {
	s.push(new(big.Int).Or(s.pop(), s.pop()))
}

func opXor(s *stack, data []byte) {
	s.push(new(big.Int).Xor(s.pop(), s.pop()))
}

func opByte(s *stack, data []byte) {
	th, val := s.pop(), s.pop()

	res := new(big.Int)
	if th.Cmp(big.NewInt(32)) < 0 {
		byt := big.NewInt(int64(common.LeftPadBytes(val.Bytes(), 32)[th.Int64()]))

		res.Set(byt)
	} else {
		res.Set(common.BigFalse)
	}

	s.push(res)
}

func opAddmod(s *stack, data []byte) {
	x := s.pop()
	y := s.pop()
	z := s.pop()

	res := new(big.Int)
	if z.Cmp(Zero) > 0 {
		add := new(big.Int).Add(x, y)
		res.Mod(add, z)
		res = U256(res)
	}

	s.push(res)
}

func opMulmod(s *stack, data []byte) {
	x := s.pop()
	y := s.pop()
	z := s.pop()

	res := new(big.Int)
	if z.Cmp(Zero) > 0 {
		res.Mul(x, y)
		res.Mod(res, z)

		res = U256(res)
	}

	s.push(res)
}
