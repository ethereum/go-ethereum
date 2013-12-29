package main

import (
  "math/big"
)

/*
 * Returns the power of two integers
 */
func BigPow(a,b int) *big.Int {
  c := new(big.Int)
  c.Exp(big.NewInt(int64(a)), big.NewInt(int64(b)), big.NewInt(0))

  return c
}

/*
 * Like big.NewInt(uint64); this takes a string instead.
 */
func Big(num string) *big.Int {
  n := new(big.Int)
  n.SetString(num, 0)

  return n
}

