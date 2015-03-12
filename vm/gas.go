package vm

import "math/big"

type req struct {
	stack int
	gas   *big.Int
}

var _baseCheck = map[OpCode]req{
	//       Req stack  Gas price
	ADD:          {2, GasFastestStep},
	LT:           {2, GasFastestStep},
	GT:           {2, GasFastestStep},
	SLT:          {2, GasFastestStep},
	SGT:          {2, GasFastestStep},
	EQ:           {2, GasFastestStep},
	ISZERO:       {1, GasFastestStep},
	SUB:          {2, GasFastestStep},
	AND:          {2, GasFastestStep},
	OR:           {2, GasFastestStep},
	XOR:          {2, GasFastestStep},
	NOT:          {1, GasFastestStep},
	BYTE:         {2, GasFastestStep},
	CALLDATALOAD: {1, GasFastestStep},
	CALLDATACOPY: {3, GasFastestStep},
	MLOAD:        {1, GasFastestStep},
	MSTORE:       {2, GasFastestStep},
	MSTORE8:      {2, GasFastestStep},
	CODECOPY:     {3, GasFastestStep},
	MUL:          {2, GasFastStep},
	DIV:          {2, GasFastStep},
	SDIV:         {2, GasFastStep},
	MOD:          {2, GasFastStep},
	SMOD:         {2, GasFastStep},
	SIGNEXTEND:   {2, GasFastStep},
	ADDMOD:       {3, GasMidStep},
	MULMOD:       {3, GasMidStep},
	JUMP:         {1, GasMidStep},
	JUMPI:        {2, GasSlowStep},
	EXP:          {2, GasSlowStep},
	ADDRESS:      {0, GasQuickStep},
	ORIGIN:       {0, GasQuickStep},
	CALLER:       {0, GasQuickStep},
	CALLVALUE:    {0, GasQuickStep},
	CODESIZE:     {0, GasQuickStep},
	GASPRICE:     {0, GasQuickStep},
	COINBASE:     {0, GasQuickStep},
	TIMESTAMP:    {0, GasQuickStep},
	NUMBER:       {0, GasQuickStep},
	CALLDATASIZE: {0, GasQuickStep},
	DIFFICULTY:   {0, GasQuickStep},
	GASLIMIT:     {0, GasQuickStep},
	POP:          {0, GasQuickStep},
	PC:           {0, GasQuickStep},
	MSIZE:        {0, GasQuickStep},
	GAS:          {0, GasQuickStep},
	BLOCKHASH:    {1, GasExtStep},
	BALANCE:      {0, GasExtStep},
	EXTCODESIZE:  {1, GasExtStep},
	EXTCODECOPY:  {4, GasExtStep},
	SLOAD:        {1, GasStorageGet},
	SSTORE:       {2, Zero},
	SHA3:         {1, GasSha3Base},
	CREATE:       {3, GasCreate},
	CALL:         {7, GasCall},
	CALLCODE:     {7, GasCall},
	JUMPDEST:     {0, GasJumpDest},
	SUICIDE:      {1, Zero},
	RETURN:       {2, Zero},
}

func baseCheck(op OpCode, stack *stack, gas *big.Int) {
	if r, ok := _baseCheck[op]; ok {
		stack.require(r.stack)

		gas.Add(gas, r.gas)
	}
}

func toWordSize(size *big.Int) *big.Int {
	tmp := new(big.Int)
	tmp.Add(size, u256(31))
	tmp.Div(tmp, u256(32))
	return tmp
}
