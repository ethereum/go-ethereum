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

func baseCheck(op OpCode, stack *stack, gas *big.Int) error {
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

		if r.stackPush && len(stack.data)-r.stackPop > int(params.StackLimit.Int64()) {
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
	stackPush bool
}

var _baseCheck = map[OpCode]req{
	// opcode  |  stack pop | gas price | stack push
	ADD:          {2, GasFastestStep, true},
	LT:           {2, GasFastestStep, true},
	GT:           {2, GasFastestStep, true},
	SLT:          {2, GasFastestStep, true},
	SGT:          {2, GasFastestStep, true},
	EQ:           {2, GasFastestStep, true},
	ISZERO:       {1, GasFastestStep, true},
	SUB:          {2, GasFastestStep, true},
	AND:          {2, GasFastestStep, true},
	OR:           {2, GasFastestStep, true},
	XOR:          {2, GasFastestStep, true},
	NOT:          {1, GasFastestStep, true},
	BYTE:         {2, GasFastestStep, true},
	CALLDATALOAD: {1, GasFastestStep, true},
	CALLDATACOPY: {3, GasFastestStep, true},
	MLOAD:        {1, GasFastestStep, true},
	MSTORE:       {2, GasFastestStep, false},
	MSTORE8:      {2, GasFastestStep, false},
	CODECOPY:     {3, GasFastestStep, false},
	MUL:          {2, GasFastStep, true},
	DIV:          {2, GasFastStep, true},
	SDIV:         {2, GasFastStep, true},
	MOD:          {2, GasFastStep, true},
	SMOD:         {2, GasFastStep, true},
	SIGNEXTEND:   {2, GasFastStep, true},
	ADDMOD:       {3, GasMidStep, true},
	MULMOD:       {3, GasMidStep, true},
	JUMP:         {1, GasMidStep, false},
	JUMPI:        {2, GasSlowStep, false},
	EXP:          {2, GasSlowStep, true},
	ADDRESS:      {0, GasQuickStep, true},
	ORIGIN:       {0, GasQuickStep, true},
	CALLER:       {0, GasQuickStep, true},
	CALLVALUE:    {0, GasQuickStep, true},
	CODESIZE:     {0, GasQuickStep, true},
	GASPRICE:     {0, GasQuickStep, true},
	COINBASE:     {0, GasQuickStep, true},
	TIMESTAMP:    {0, GasQuickStep, true},
	NUMBER:       {0, GasQuickStep, true},
	CALLDATASIZE: {0, GasQuickStep, true},
	DIFFICULTY:   {0, GasQuickStep, true},
	GASLIMIT:     {0, GasQuickStep, true},
	POP:          {1, GasQuickStep, false},
	PC:           {0, GasQuickStep, true},
	MSIZE:        {0, GasQuickStep, true},
	GAS:          {0, GasQuickStep, true},
	BLOCKHASH:    {1, GasExtStep, true},
	BALANCE:      {1, GasExtStep, true},
	EXTCODESIZE:  {1, GasExtStep, true},
	EXTCODECOPY:  {4, GasExtStep, false},
	SLOAD:        {1, params.SloadGas, true},
	SSTORE:       {2, Zero, false},
	SHA3:         {2, params.Sha3Gas, true},
	CREATE:       {3, params.CreateGas, true},
	CALL:         {7, params.CallGas, true},
	CALLCODE:     {7, params.CallGas, true},
	JUMPDEST:     {0, params.JumpdestGas, false},
	SUICIDE:      {1, Zero, false},
	RETURN:       {2, Zero, false},
	PUSH1:        {0, GasFastestStep, true},
	DUP1:         {0, Zero, true},
}
