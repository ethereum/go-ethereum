package vm

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
)

var vmlogger = logger.NewLogger("VM")

// Global Debug flag indicating Debug VM (full logging)
var Debug bool

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

	Pow256 = common.BigPow(2, 256)

	LogTyPretty byte = 0x1
	LogTyDiff   byte = 0x2

	U256 = common.U256
	S256 = common.S256

	Zero = common.Big0
)

const MaxCallDepth = 1025

func calcMemSize(off, l *big.Int) *big.Int {
	if l.Cmp(common.Big0) == 0 {
		return common.Big0
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

func getData(data []byte, start, size uint64) []byte {
	x := uint64(math.Min(float64(start), float64(len(data))))
	y := uint64(math.Min(float64(x+size), float64(len(data))))

	return common.RightPadBytes(data[x:y], int(size))
}
