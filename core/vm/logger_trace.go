package vm

import (
	"github.com/scroll-tech/go-ethereum/common"
)

type traceFunc func(l *StructLogger, scope *ScopeContext) error

var (
	// OpcodeExecs the map to load opcodes' trace funcs.
	OpcodeExecs = map[OpCode][]traceFunc{
		CALL:         {traceToAddressCode, traceLastNAddressCode(1), traceContractAccount, traceLastNAddressAccount(1)}, // contract account is the caller, stack.nth_last(1) is the callee's address
		CALLCODE:     {traceToAddressCode, traceLastNAddressCode(1), traceContractAccount, traceLastNAddressAccount(1)}, // contract account is the caller, stack.nth_last(1) is the callee's address
		DELEGATECALL: {traceToAddressCode, traceLastNAddressCode(1)},
		STATICCALL:   {traceToAddressCode, traceLastNAddressCode(1), traceLastNAddressAccount(1)},
		SELFDESTRUCT: {traceContractAccount, traceLastNAddressAccount(0)},
		SELFBALANCE:  {traceContractAccount},
		BALANCE:      {traceLastNAddressAccount(0)},
		EXTCODEHASH:  {traceLastNAddressAccount(0)},
		EXTCODESIZE:  {traceLastNAddressAccount(0)},
		EXTCODECOPY:  {traceLastNAddressCode(0)},
	}
)

// traceToAddressCode gets tx.to address’s code
func traceToAddressCode(l *StructLogger, scope *ScopeContext) error {
	if l.env.To == nil {
		return nil
	}
	traceCodeWithAddress(l, *l.env.To)
	return nil
}

// traceLastNAddressCode
func traceLastNAddressCode(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}
		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		traceCodeWithAddress(l, address)
		l.statesAffected[address] = struct{}{}
		return nil
	}
}

func traceCodeWithAddress(l *StructLogger, address common.Address) {
	code := l.env.StateDB.GetCode(address)
	keccakCodeHash := l.env.StateDB.GetKeccakCodeHash(address)
	poseidonCodeHash := l.env.StateDB.GetPoseidonCodeHash(address)
	codeSize := l.env.StateDB.GetCodeSize(address)
	l.bytecodes[poseidonCodeHash] = CodeInfo{
		codeSize,
		keccakCodeHash,
		poseidonCodeHash,
		code,
	}
}

// traceContractAccount gets the contract's account
func traceContractAccount(l *StructLogger, scope *ScopeContext) error {
	l.statesAffected[scope.Contract.Address()] = struct{}{}

	return nil
}

// traceLastNAddressAccount returns func about the last N's address account.
func traceLastNAddressAccount(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}

		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		l.statesAffected[address] = struct{}{}

		return nil
	}
}
