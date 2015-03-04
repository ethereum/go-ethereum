package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
)

var vmlogger = logger.NewLogger("VM")

type Type byte

const (
	StdVmTy Type = iota
	JitVmTy

	MaxVmTy
)

func NewVm(env Environment) VirtualMachine {
	switch env.VmType() {
	case JitVmTy:
		return NewJitVm(env)
	default:
		vmlogger.Infoln("unsupported vm type %d", env.VmType())
		fallthrough
	case StdVmTy:
		return New(env)
	}
}

var (
	GasQuickStep   = big.NewInt(2)
	GasFastestStep = big.NewInt(3)
	GasFastStep    = big.NewInt(5)
	GasMidStep     = big.NewInt(8)
	GasSlowStep    = big.NewInt(10)
	GasExtStep     = big.NewInt(20)

	GasStorageGet        = big.NewInt(50)
	GasStorageAdd        = big.NewInt(20000)
	GasStorageMod        = big.NewInt(5000)
	GasLogBase           = big.NewInt(375)
	GasLogTopic          = big.NewInt(375)
	GasLogByte           = big.NewInt(8)
	GasCreate            = big.NewInt(32000)
	GasCreateByte        = big.NewInt(200)
	GasCall              = big.NewInt(40)
	GasCallValueTransfer = big.NewInt(9000)
	GasStipend           = big.NewInt(2300)
	GasCallNewAccount    = big.NewInt(25000)
	GasReturn            = big.NewInt(0)
	GasStop              = big.NewInt(0)
	GasJumpDest          = big.NewInt(1)

	RefundStorage = big.NewInt(15000)
	RefundSuicide = big.NewInt(24000)

	GasMemWord           = big.NewInt(3)
	GasQuadCoeffDenom    = big.NewInt(512)
	GasContractByte      = big.NewInt(200)
	GasTransaction       = big.NewInt(21000)
	GasTxDataNonzeroByte = big.NewInt(68)
	GasTxDataZeroByte    = big.NewInt(4)
	GasTx                = big.NewInt(21000)
	GasExp               = big.NewInt(10)
	GasExpByte           = big.NewInt(10)

	GasSha3Base     = big.NewInt(30)
	GasSha3Word     = big.NewInt(6)
	GasSha256Base   = big.NewInt(60)
	GasSha256Word   = big.NewInt(12)
	GasRipemdBase   = big.NewInt(600)
	GasRipemdWord   = big.NewInt(12)
	GasEcrecover    = big.NewInt(3000)
	GasIdentityBase = big.NewInt(15)
	GasIdentityWord = big.NewInt(3)
	GasCopyWord     = big.NewInt(3)

	Pow256 = ethutil.BigPow(2, 256)

	LogTyPretty byte = 0x1
	LogTyDiff   byte = 0x2

	U256 = ethutil.U256
	S256 = ethutil.S256

	Zero = ethutil.Big0
)

const MaxCallDepth = 1025

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
