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
		CALL:         {traceToAddressCode, traceLastNAddressCode(1), traceCaller, traceLastNAddressAccount(1)},
		CALLCODE:     {traceToAddressCode, traceLastNAddressCode(1), traceCaller, traceLastNAddressAccount(1)},
		DELEGATECALL: {traceToAddressCode, traceLastNAddressCode(1)},
		STATICCALL:   {traceToAddressCode, traceLastNAddressCode(1), traceLastNAddressAccount(1)},
		CREATE:       {}, // sender is already recorded in ExecutionResult, callee is recorded in CaptureEnter&CaptureExit
		CREATE2:      {}, // sender is already recorded in ExecutionResult, callee is recorded in CaptureEnter&CaptureExit
		SLOAD:        {}, // trace storage in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SSTORE:       {}, // trace storage in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SELFDESTRUCT: {traceContractAccount, traceLastNAddressAccount(0)},
		SELFBALANCE:  {traceContractAccount},
		BALANCE:      {traceLastNAddressAccount(0)},
		EXTCODEHASH:  {traceLastNAddressAccount(0)},
	}
)

// traceToAddressCode gets tx.to address’s code
func traceToAddressCode(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if l.env.To == nil {
		return nil
	}
	code := l.env.StateDB.GetCode(*l.env.To)
	extraData.CodeList = append(extraData.CodeList, code)
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
		extraData.CodeList = append(extraData.CodeList, code)
		return nil
	}
}

// traceStorage get contract's storage at storage_address
func traceStorage(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if scope.Stack.len() == 0 {
		return nil
	}
	key := common.Hash(scope.Stack.peek().Bytes32())
	storage, err := getWrappedAccountForStorage(l, scope.Contract.Address(), key)
	if err == nil {
		extraData.StateList = append(extraData.StateList, storage)
	}
	return err
}

// traceContractAccount gets the contract's account
func traceContractAccount(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	// Get account state.
	state, err := getWrappedAccountForAddr(l, scope.Contract.Address())
	if err == nil {
		extraData.StateList = append(extraData.StateList, state)
		l.statesAffected[scope.Contract.Address()] = struct{}{}
	}
	return err
}

// traceLastNAddressAccount returns func about the last N's address account.
func traceLastNAddressAccount(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}

		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		state, err := getWrappedAccountForAddr(l, address)
		if err == nil {
			extraData.StateList = append(extraData.StateList, state)
			l.statesAffected[scope.Contract.Address()] = struct{}{}
		}
		return err
	}
}

// traceCaller gets caller address's account.
func traceCaller(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	address := scope.Contract.CallerAddress
	state, err := getWrappedAccountForAddr(l, address)
	if err == nil {
		extraData.StateList = append(extraData.StateList, state)
		l.statesAffected[scope.Contract.Address()] = struct{}{}
	}
	return err
}

// StorageWrapper will be empty
func getWrappedAccountForAddr(l *StructLogger, address common.Address) (*types.AccountWrapper, error) {
	return &types.AccountWrapper{
		Address:  address,
		Nonce:    l.env.StateDB.GetNonce(address),
		Balance:  (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		CodeHash: l.env.StateDB.GetCodeHash(address),
	}, nil
}

func getWrappedAccountForStorage(l *StructLogger, address common.Address, key common.Hash) (*types.AccountWrapper, error) {
	return &types.AccountWrapper{
		Address:  address,
		Nonce:    l.env.StateDB.GetNonce(address),
		Balance:  (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		CodeHash: l.env.StateDB.GetCodeHash(address),
		Storage: &types.StorageWrapper{
			Key:   key.String(),
			Value: l.env.StateDB.GetState(address, key).String(),
		},
	}, nil
}
