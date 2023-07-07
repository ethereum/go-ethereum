package vm

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type traceFunc func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error

var (
	// OpcodeExecs the map to load opcodes' trace funcs.
	OpcodeExecs = map[OpCode][]traceFunc{
		CALL:         {traceToAddressCode, traceLastNAddressCode(1), traceContractAccount, traceLastNAddressAccount(1)}, // contract account is the caller, stack.nth_last(1) is the callee's address
		CALLCODE:     {traceToAddressCode, traceLastNAddressCode(1), traceContractAccount, traceLastNAddressAccount(1)}, // contract account is the caller, stack.nth_last(1) is the callee's address
		DELEGATECALL: {traceToAddressCode, traceLastNAddressCode(1)},
		STATICCALL:   {traceToAddressCode, traceLastNAddressCode(1), traceLastNAddressAccount(1)},
		CREATE:       {}, // caller is already recorded in ExtraData.Caller, callee is recorded in CaptureEnter&CaptureExit
		CREATE2:      {}, // caller is already recorded in ExtraData.Caller, callee is recorded in CaptureEnter&CaptureExit
		SLOAD:        {}, // trace storage in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SSTORE:       {}, // trace storage in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SELFDESTRUCT: {traceContractAccount, traceLastNAddressAccount(0)},
		SELFBALANCE:  {traceContractAccount},
		BALANCE:      {traceLastNAddressAccount(0)},
		EXTCODEHASH:  {traceLastNAddressAccount(0)},
		CODESIZE:     {traceContractCode},
		CODECOPY:     {traceContractCode},
		EXTCODESIZE:  {traceLastNAddressCode(0)},
		EXTCODECOPY:  {traceLastNAddressCode(0)},
	}
)

// traceToAddressCode gets tx.to address’s code
func traceToAddressCode(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if l.env.To == nil {
		return nil
	}
	code := l.env.StateDB.GetCode(*l.env.To)
	extraData.CodeList = append(extraData.CodeList, hexutil.Encode(code))
	return nil
}

// traceLastNAddressCode
func traceLastNAddressCode(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}
		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		code := l.env.StateDB.GetCode(address)
		extraData.CodeList = append(extraData.CodeList, hexutil.Encode(code))
		l.statesAffected[address] = struct{}{}
		return nil
	}
}

// traceContractCode gets the contract's code
func traceContractCode(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	code := l.env.StateDB.GetCode(scope.Contract.Address())
	extraData.CodeList = append(extraData.CodeList, hexutil.Encode(code))
	return nil
}

// traceStorage get contract's storage at storage_address
func traceStorage(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if scope.Stack.len() == 0 {
		return nil
	}
	key := common.Hash(scope.Stack.peek().Bytes32())
	storage := getWrappedAccountForStorage(l, scope.Contract.Address(), key)
	extraData.StateList = append(extraData.StateList, storage)

	return nil
}

// traceContractAccount gets the contract's account
func traceContractAccount(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	// Get account state.
	state := getWrappedAccountForAddr(l, scope.Contract.Address())
	extraData.StateList = append(extraData.StateList, state)
	l.statesAffected[scope.Contract.Address()] = struct{}{}

	return nil
}

// traceLastNAddressAccount returns func about the last N's address account.
func traceLastNAddressAccount(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}

		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		state := getWrappedAccountForAddr(l, address)
		extraData.StateList = append(extraData.StateList, state)
		l.statesAffected[address] = struct{}{}

		return nil
	}
}

// StorageWrapper will be empty
func getWrappedAccountForAddr(l *StructLogger, address common.Address) *types.AccountWrapper {
	return &types.AccountWrapper{
		Address:          address,
		Nonce:            l.env.StateDB.GetNonce(address),
		Balance:          (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		KeccakCodeHash:   l.env.StateDB.GetKeccakCodeHash(address),
		PoseidonCodeHash: l.env.StateDB.GetPoseidonCodeHash(address),
		CodeSize:         l.env.StateDB.GetCodeSize(address),
	}
}

func getWrappedAccountForStorage(l *StructLogger, address common.Address, key common.Hash) *types.AccountWrapper {
	return &types.AccountWrapper{
		Address:          address,
		Nonce:            l.env.StateDB.GetNonce(address),
		Balance:          (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		KeccakCodeHash:   l.env.StateDB.GetKeccakCodeHash(address),
		PoseidonCodeHash: l.env.StateDB.GetPoseidonCodeHash(address),
		CodeSize:         l.env.StateDB.GetCodeSize(address),
		Storage: &types.StorageWrapper{
			Key:   key.String(),
			Value: l.env.StateDB.GetState(address, key).String(),
		},
	}
}

func getCodeForAddr(l *StructLogger, address common.Address) []byte {
	return l.env.StateDB.GetCode(address)
}
