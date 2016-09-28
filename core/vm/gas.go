package vm

import (
	"fmt"
	gmath "math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

var (
	maxInt63  = new(big.Int).Exp(big.NewInt(2), big.NewInt(63), big.NewInt(0))
	maxIntCap = new(big.Int).Sub(maxInt63, big.NewInt(1))
)

var (
	StackLimit64            = params.StackLimit.Uint64()
	GasQuickStep64   uint64 = 2
	GasFastestStep64 uint64 = 3
	GasFastStep64    uint64 = 5
	GasMidStep64     uint64 = 8
	GasSlowStep64    uint64 = 10
	GasExtStep64     uint64 = 20

	GasReturn64 uint64 = 0
	GasStop64   uint64 = 0

	GasContractByte64      uint64 = 200
	LogGas64                      = params.LogGas.Uint64()
	LogTopicGas64                 = params.LogTopicGas.Uint64()
	LogDataGas64                  = params.LogDataGas.Uint64()
	ExpByteGas64                  = params.ExpByteGas.Uint64()
	SstoreSetGas64                = params.SstoreSetGas.Uint64()
	SstoreClearGas64              = params.SstoreClearGas.Uint64()
	SstoreResetGas64              = params.SstoreResetGas.Uint64()
	KeccakWordGas64               = params.Sha3WordGas.Uint64()
	CopyGas64                     = params.CopyGas.Uint64()
	CallNewAccountGas64           = params.CallNewAccountGas.Uint64()
	CallValueTransferGas64        = params.CallValueTransferGas.Uint64()
	MemoryGas64                   = params.MemoryGas.Uint64()
	QuadCoeffDiv64                = params.QuadCoeffDiv.Uint64()
)

// casts a arbitrary number to the amount of words (sets of 32 bytes)
func toWordSize(size uint64) uint64 {
	return (size + 31) / 32
}

// calculates the memory size required for a step
func calcMemSize(off, l *big.Int) (uint64, bool) {
	if l.Cmp(common.Big0) == 0 {
		return 0, false
	}
	size := new(big.Int).Add(off, l)
	if size.Cmp(maxIntCap) > 0 {
		return 0, true
	}
	return size.Uint64(), false
}

// calculates the quadratic gas
// TODO this function requires guarding of overflows
func calcQuadMemGas(mem *Memory, newMemSize uint64) (uint64, bool) {
	oldTotalFee := mem.cost
	if newMemSize > 0 {
		newMemSizeWords := toWordSize(newMemSize)
		newMemSize = newMemSizeWords * 32

		if newMemSize > uint64(mem.Len()) {
			pow := uint64(gmath.Pow(float64(newMemSizeWords), 2))
			linCoef := newMemSizeWords * MemoryGas64
			quadCoef := pow / QuadCoeffDiv64
			newTotalFee := linCoef + quadCoef

			fee := newTotalFee - oldTotalFee
			mem.cost = newTotalFee

			return fee, false
		}
	}
	return 0, false
}

// baseCalc is the same as baseCheck except it doesn't do the look up in the
// gas table. This is done during compilation instead.
func baseCalc(instr instruction, stack *Stack) (uint64, error) {
	err := stack.require(instr.spop)
	if err != nil {
		return 0, err
	}

	if instr.spush > 0 && stack.len()-instr.spop+instr.spush > int(StackLimit64) {
		return 0, fmt.Errorf("stack limit reached %d (%d)", stack.len(), StackLimit64)
	}

	// 0 on gas means no base calculation
	if instr.gas == 0 {
		return 0, nil
	}

	return instr.gas, nil
}

type req struct {
	stackPop  int
	gas       uint64
	stackPush int
}

var _baseCheck = map[OpCode]req{
	// opcode  |  stack pop | gas price | stack push
	ADD:          {2, GasFastestStep64, 1},
	LT:           {2, GasFastestStep64, 1},
	GT:           {2, GasFastestStep64, 1},
	SLT:          {2, GasFastestStep64, 1},
	SGT:          {2, GasFastestStep64, 1},
	EQ:           {2, GasFastestStep64, 1},
	ISZERO:       {1, GasFastestStep64, 1},
	SUB:          {2, GasFastestStep64, 1},
	AND:          {2, GasFastestStep64, 1},
	OR:           {2, GasFastestStep64, 1},
	XOR:          {2, GasFastestStep64, 1},
	NOT:          {1, GasFastestStep64, 1},
	BYTE:         {2, GasFastestStep64, 1},
	CALLDATALOAD: {1, GasFastestStep64, 1},
	CALLDATACOPY: {3, GasFastestStep64, 1},
	MLOAD:        {1, GasFastestStep64, 1},
	MSTORE:       {2, GasFastestStep64, 0},
	MSTORE8:      {2, GasFastestStep64, 0},
	CODECOPY:     {3, GasFastestStep64, 0},
	MUL:          {2, GasFastStep64, 1},
	DIV:          {2, GasFastStep64, 1},
	SDIV:         {2, GasFastStep64, 1},
	MOD:          {2, GasFastStep64, 1},
	SMOD:         {2, GasFastStep64, 1},
	SIGNEXTEND:   {2, GasFastStep64, 1},
	ADDMOD:       {3, GasMidStep64, 1},
	MULMOD:       {3, GasMidStep64, 1},
	JUMP:         {1, GasMidStep64, 0},
	JUMPI:        {2, GasSlowStep64, 0},
	EXP:          {2, GasSlowStep64, 1},
	ADDRESS:      {0, GasQuickStep64, 1},
	ORIGIN:       {0, GasQuickStep64, 1},
	CALLER:       {0, GasQuickStep64, 1},
	CALLVALUE:    {0, GasQuickStep64, 1},
	CODESIZE:     {0, GasQuickStep64, 1},
	GASPRICE:     {0, GasQuickStep64, 1},
	COINBASE:     {0, GasQuickStep64, 1},
	TIMESTAMP:    {0, GasQuickStep64, 1},
	NUMBER:       {0, GasQuickStep64, 1},
	CALLDATASIZE: {0, GasQuickStep64, 1},
	DIFFICULTY:   {0, GasQuickStep64, 1},
	GASLIMIT:     {0, GasQuickStep64, 1},
	POP:          {1, GasQuickStep64, 0},
	PC:           {0, GasQuickStep64, 1},
	MSIZE:        {0, GasQuickStep64, 1},
	GAS:          {0, GasQuickStep64, 1},
	BLOCKHASH:    {1, GasExtStep64, 1},
	BALANCE:      {1, GasExtStep64, 1},
	EXTCODESIZE:  {1, GasExtStep64, 1},
	EXTCODECOPY:  {4, GasExtStep64, 0},
	SLOAD:        {1, params.SloadGas.Uint64(), 1},
	SSTORE:       {2, 0, 0},
	SHA3:         {2, params.Sha3Gas.Uint64(), 1},
	CREATE:       {3, params.CreateGas.Uint64(), 1},
	CALL:         {7, params.CallGas.Uint64(), 1},
	CALLCODE:     {7, params.CallGas.Uint64(), 1},
	DELEGATECALL: {6, params.CallGas.Uint64(), 1},
	JUMPDEST:     {0, params.JumpdestGas.Uint64(), 0},
	SUICIDE:      {1, 0, 0},
	RETURN:       {2, 0, 0},
	PUSH1:        {0, GasFastestStep64, 1},
	DUP1:         {0, 0, 1},
}
