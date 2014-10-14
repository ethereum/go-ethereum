package ethvm

import (
	"math/big"

	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
)

var vmlogger = ethlog.NewLogger("VM")

type Type int

const (
	StandardVmTy Type = iota
	DebugVmTy

	MaxVmTy
)

var (
	GasStep    = big.NewInt(1)
	GasSha     = big.NewInt(20)
	GasSLoad   = big.NewInt(20)
	GasSStore  = big.NewInt(100)
	GasBalance = big.NewInt(20)
	GasCreate  = big.NewInt(100)
	GasCall    = big.NewInt(20)
	GasMemory  = big.NewInt(1)
	GasData    = big.NewInt(5)
	GasTx      = big.NewInt(500)

	Pow256 = ethutil.BigPow(2, 256)

	LogTyPretty byte = 0x1
	LogTyDiff   byte = 0x2

	To256 = ethutil.To256
)

const MaxCallDepth = 1024

func calcMemSize(off, l *big.Int) *big.Int {
	if l.Cmp(ethutil.Big0) == 0 {
		return ethutil.Big0
	}

	return new(big.Int).Add(off, l)
}

// Simple helper
func u256(n int64) *big.Int {
	return big.NewInt(n)
}

// Mainly used for print variables and passing to Print*
func toValue(val *big.Int) interface{} {
	// Let's assume a string on right padded zero's
	b := val.Bytes()
	if b[0] != 0 && b[len(b)-1] == 0x0 && b[len(b)-2] == 0x0 {
		return string(b)
	}

	return val
}
