package vm

import "math/big"

type req struct {
	stack int
	gas   *big.Int
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
)

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
