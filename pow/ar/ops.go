package ar

import "math/big"

const lenops int64 = 9

type OpsFunc func(a, b *big.Int) *big.Int

var ops [lenops]OpsFunc

func init() {
	ops[0] = Add
	ops[1] = Mul
	ops[2] = Mod
	ops[3] = Xor
	ops[4] = And
	ops[5] = Or
	ops[6] = Sub1
	ops[7] = XorSub
	ops[8] = Rsh
}

func Add(x, y *big.Int) *big.Int {
	return new(big.Int).Add(x, y)
}
func Mul(x, y *big.Int) *big.Int {
	return new(big.Int).Mul(x, y)
}
func Mod(x, y *big.Int) *big.Int {
	return new(big.Int).Mod(x, y)
}
func Xor(x, y *big.Int) *big.Int {
	return new(big.Int).Xor(x, y)
}
func And(x, y *big.Int) *big.Int {
	return new(big.Int).And(x, y)
}
func Or(x, y *big.Int) *big.Int {
	return new(big.Int).Or(x, y)
}
func Sub1(x, y *big.Int) *big.Int {
	a := big.NewInt(-1)
	a.Sub(a, x)

	return a
}
func XorSub(x, y *big.Int) *big.Int {
	t := Sub1(x, nil)

	return t.Xor(t, y)
}
func Rsh(x, y *big.Int) *big.Int {
	return new(big.Int).Rsh(x, uint(y.Uint64()%64))
}
