package vm

import (
	"errors"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type traceFunc func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error

var (
	// OpcodeExecs the map to load opcodes' trace funcs.
	OpcodeExecs = map[OpCode][]traceFunc{
		CALL:         {traceToAddressCodeHash, traceLastNAddressCodeHash(1), traceCallerProof, traceLastNAddressProof(1)},
		CALLCODE:     {traceToAddressCodeHash, traceLastNAddressCodeHash(1), traceCallerProof, traceLastNAddressProof(1)},
		DELEGATECALL: {traceToAddressCodeHash, traceLastNAddressCodeHash(1)},
		STATICCALL:   {traceToAddressCodeHash, traceLastNAddressCodeHash(1)},
		CREATE:       {traceSenderAddress, traceCreatedContractProof, traceNonce},
		CREATE2:      {traceSenderAddress, traceCreatedContractProof},
		SSTORE:       {traceStorageProof},
		SLOAD:        {traceStorageProof},
		SELFDESTRUCT: {traceContractProof, traceLastNAddressProof(0)},
		SELFBALANCE:  {traceContractProof},
		BALANCE:      {traceLastNAddressProof(0)},
		EXTCODEHASH:  {traceLastNAddressProof(0)},
	}
)

// traceToAddressCodeHash gets tx.to address’s code_hash
func traceToAddressCodeHash(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if l.env.To == nil {
		return nil
	}
	codeHash := l.env.StateDB.GetCodeHash(*l.env.To)
	extraData.CodeHashList = append(extraData.CodeHashList, codeHash)
	return nil
}

// traceLastNAddressCodeHash
func traceLastNAddressCodeHash(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}
		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		codeHash := l.env.StateDB.GetCodeHash(address)
		extraData.CodeHashList = append(extraData.CodeHashList, codeHash)
		return nil
	}
}

// traceSenderAddress gets sender address
func traceSenderAddress(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	extraData.From = &l.env.Origin
	return nil
}

// traceNonce gets sender nonce
func traceNonce(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	nonce := l.env.StateDB.GetNonce(l.env.Origin)
	extraData.Nonce = &nonce
	return nil
}

// traceStorageProof get contract's storage proof at storage_address
func traceStorageProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if scope.Stack.len() == 0 {
		return nil
	}
	address := common.Hash(scope.Stack.peek().Bytes32())
	contract := scope.Contract
	// Get storage proof.
	storageProof, err := l.env.StateDB.GetStorageProof(contract.Address(), address)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, encodeProof(storageProof))
	}
	return err
}

// traceContractProof gets the contract's account proof
func traceContractProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	// Get account proof.
	proof, err := l.env.StateDB.GetProof(scope.Contract.Address())
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, encodeProof(proof))
	}
	return err
}

/// traceCreatedContractProof get created contract address’s accountProof
func traceCreatedContractProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	stack := scope.Stack
	if stack.len() < 1 {
		return nil
	}
	stackvalue := stack.peek()
	if stackvalue.IsZero() {
		return errors.New("can't get created contract address from stack")
	}
	address := common.BytesToAddress(stackvalue.Bytes())
	proof, err := l.env.StateDB.GetProof(address)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, encodeProof(proof))
	}
	return err
}

// traceLastNAddressProof returns func about the last N's address proof.
func traceLastNAddressProof(n int) traceFunc {
	return func(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
		stack := scope.Stack
		if stack.len() <= n {
			return nil
		}

		address := common.Address(stack.data[stack.len()-1-n].Bytes20())
		proof, err := l.env.StateDB.GetProof(address)
		if err == nil {
			extraData.ProofList = append(extraData.ProofList, encodeProof(proof))
		}
		return err
	}
}

// traceCallerProof gets caller address's proof.
func traceCallerProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	address := scope.Contract.CallerAddress
	proof, err := l.env.StateDB.GetProof(address)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, encodeProof(proof))
	}
	return err
}

func encodeProof(proof [][]byte) (res []string) {
	if len(proof) == 0 {
		return nil
	}
	for _, node := range proof {
		res = append(res, hexutil.Encode(node))
	}
	return
}
