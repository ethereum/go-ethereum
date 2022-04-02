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
		CALL:         {traceToAddressCode, traceLastNAddressCode(1), traceCallerProof, traceLastNAddressProof(1)},
		CALLCODE:     {traceToAddressCode, traceLastNAddressCode(1), traceCallerProof, traceLastNAddressProof(1)},
		DELEGATECALL: {traceToAddressCode, traceLastNAddressCode(1)},
		STATICCALL:   {traceToAddressCode, traceLastNAddressCode(1)},
		CREATE:       {traceCreatedContractProof}, // sender's wrapped_proof is already recorded in BlockChain.writeBlockResult
		CREATE2:      {traceCreatedContractProof}, // sender's wrapped_proof is already recorded in BlockChain.writeBlockResult
		SLOAD:        {},                          // record storage_proof in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SSTORE:       {},                          // record storage_proof in `captureState` instead of here, to handle `l.cfg.DisableStorage` flag
		SELFDESTRUCT: {traceContractProof, traceLastNAddressProof(0)},
		SELFBALANCE:  {traceContractProof},
		BALANCE:      {traceLastNAddressProof(0)},
		EXTCODEHASH:  {traceLastNAddressProof(0)},
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

// traceStorageProof get contract's storage proof at storage_address
func traceStorageProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	if scope.Stack.len() == 0 {
		return nil
	}
	key := common.Hash(scope.Stack.peek().Bytes32())
	proof, err := getWrappedProofForStorage(l, scope.Contract.Address(), key)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, proof)
	}
	return err
}

// traceContractProof gets the contract's account proof
func traceContractProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	// Get account proof.
	proof, err := getWrappedProofForAddr(l, scope.Contract.Address())
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, proof)
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
	proof, err := getWrappedProofForAddr(l, address)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, proof)
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
		proof, err := getWrappedProofForAddr(l, address)
		if err == nil {
			extraData.ProofList = append(extraData.ProofList, proof)
		}
		return err
	}
}

// traceCallerProof gets caller address's proof.
func traceCallerProof(l *StructLogger, scope *ScopeContext, extraData *types.ExtraData) error {
	address := scope.Contract.CallerAddress
	proof, err := getWrappedProofForAddr(l, address)
	if err == nil {
		extraData.ProofList = append(extraData.ProofList, proof)
	}
	return err
}

// StorageProofWrapper will be empty
func getWrappedProofForAddr(l *StructLogger, address common.Address) (*types.AccountProofWrapper, error) {
	proof, err := l.env.StateDB.GetProof(address)
	if err != nil {
		return nil, err
	}

	return &types.AccountProofWrapper{
		Address:  address,
		Nonce:    l.env.StateDB.GetNonce(address),
		Balance:  (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		CodeHash: l.env.StateDB.GetCodeHash(address),
		Proof:    encodeProof(proof),
	}, nil
}

func getWrappedProofForStorage(l *StructLogger, address common.Address, key common.Hash) (*types.AccountProofWrapper, error) {
	proof, err := l.env.StateDB.GetProof(address)
	if err != nil {
		return nil, err
	}

	storageProof, err := l.env.StateDB.GetStorageProof(address, key)
	if err != nil {
		return nil, err
	}

	return &types.AccountProofWrapper{
		Address:  address,
		Nonce:    l.env.StateDB.GetNonce(address),
		Balance:  (*hexutil.Big)(l.env.StateDB.GetBalance(address)),
		CodeHash: l.env.StateDB.GetCodeHash(address),
		Proof:    encodeProof(proof),
		Storage: &types.StorageProofWrapper{
			Key:   key.String(),
			Value: l.env.StateDB.GetState(address, key).String(),
			Proof: encodeProof(storageProof),
		},
	}, nil
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
