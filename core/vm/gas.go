package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

var (
	GasQuickStep   = big.NewInt(2)
	GasFastestStep = big.NewInt(3)
	GasFastStep    = big.NewInt(5)
	GasMidStep     = big.NewInt(8)
	GasSlowStep    = big.NewInt(10)
	GasExtStep     = big.NewInt(20)

	GasReturn = big.NewInt(0)
	GasStop   = big.NewInt(0)

	GasContractByte = big.NewInt(200)
)

func baseCheck(op OpCode, stack *Stack, gas *big.Int) error {
	// PUSH and DUP are a bit special. They all cost the same but we do want to have checking on stack push limit
	// PUSH is also allowed to calculate the same price for all PUSHes
	// DUP requirements are handled elsewhere (except for the stack limit check)
	if op >= PUSH1 && op <= PUSH32 {
		op = PUSH1
	}
	if op >= DUP1 && op <= DUP16 {
		op = DUP1
	}

	if r, ok := _baseCheck[op]; ok {
		err := stack.require(r.stackPop)
		if err != nil {
			return err
		}

		if r.stackPush > 0 && len(stack.data)-r.stackPop+r.stackPush > int(params.StackLimit.Int64())+1 {
			return fmt.Errorf("stack limit reached %d (%d)", len(stack.data), params.StackLimit.Int64())
		}

		gas.Add(gas, r.gas)
	}
	return nil
}

func toWordSize(size *big.Int) *big.Int {
	tmp := new(big.Int)
	tmp.Add(size, u256(31))
	tmp.Div(tmp, u256(32))
	return tmp
}

type req struct {
	stackPop  int
	gas       *big.Int
	stackPush int
}

var _baseCheck = map[OpCode]req{
	// opcode  |  stack pop | gas price | stack push
	ADD:          {2, GasFastestStep, 1},
	LT:           {2, GasFastestStep, 1},
	GT:           {2, GasFastestStep, 1},
	SLT:          {2, GasFastestStep, 1},
	SGT:          {2, GasFastestStep, 1},
	EQ:           {2, GasFastestStep, 1},
	ISZERO:       {1, GasFastestStep, 1},
	SUB:          {2, GasFastestStep, 1},
	AND:          {2, GasFastestStep, 1},
	OR:           {2, GasFastestStep, 1},
	XOR:          {2, GasFastestStep, 1},
	NOT:          {1, GasFastestStep, 1},
	BYTE:         {2, GasFastestStep, 1},
	CALLDATALOAD: {1, GasFastestStep, 1},
	CALLDATACOPY: {3, GasFastestStep, 1},
	MLOAD:        {1, GasFastestStep, 1},
	MSTORE:       {2, GasFastestStep, 0},
	MSTORE8:      {2, GasFastestStep, 0},
	CODECOPY:     {3, GasFastestStep, 0},
	MUL:          {2, GasFastStep, 1},
	DIV:          {2, GasFastStep, 1},
	SDIV:         {2, GasFastStep, 1},
	MOD:          {2, GasFastStep, 1},
	SMOD:         {2, GasFastStep, 1},
	SIGNEXTEND:   {2, GasFastStep, 1},
	ADDMOD:       {3, GasMidStep, 1},
	MULMOD:       {3, GasMidStep, 1},
	JUMP:         {1, GasMidStep, 0},
	JUMPI:        {2, GasSlowStep, 0},
	EXP:          {2, GasSlowStep, 1},
	ADDRESS:      {0, GasQuickStep, 1},
	ORIGIN:       {0, GasQuickStep, 1},
	CALLER:       {0, GasQuickStep, 1},
	CALLVALUE:    {0, GasQuickStep, 1},
	CODESIZE:     {0, GasQuickStep, 1},
	GASPRICE:     {0, GasQuickStep, 1},
	COINBASE:     {0, GasQuickStep, 1},
	TIMESTAMP:    {0, GasQuickStep, 1},
	NUMBER:       {0, GasQuickStep, 1},
	CALLDATASIZE: {0, GasQuickStep, 1},
	DIFFICULTY:   {0, GasQuickStep, 1},
	GASLIMIT:     {0, GasQuickStep, 1},
	POP:          {1, GasQuickStep, 0},
	PC:           {0, GasQuickStep, 1},
	MSIZE:        {0, GasQuickStep, 1},
	GAS:          {0, GasQuickStep, 1},
	BLOCKHASH:    {1, GasExtStep, 1},
	BALANCE:      {1, GasExtStep, 1},
	EXTCODESIZE:  {1, GasExtStep, 1},
	EXTCODECOPY:  {4, GasExtStep, 0},
	SLOAD:        {1, params.SloadGas, 1},
	SSTORE:       {2, Zero, 0},
	SHA3:         {2, params.Sha3Gas, 1},
	CREATE:       {3, params.CreateGas, 1},
	CALL:         {7, params.CallGas, 1},
	CALLCODE:     {7, params.CallGas, 1},
	JUMPDEST:     {0, params.JumpdestGas, 0},
	SUICIDE:      {1, Zero, 0},
	RETURN:       {2, Zero, 0},
	PUSH1:        {0, GasFastestStep, 1},
	DUP1:         {0, Zero, 1},
}
