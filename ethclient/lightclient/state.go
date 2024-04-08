// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package lightclient

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

type proofRequest struct {
	blockNumber uint64
	address     common.Address
	storageKeys string
}

type codeRequest struct {
	blockNumber uint64
	address     common.Address
}

type lightState struct {
	client           *rpc.Client
	canonicalChain   *canonicalChain
	blocksAndHeaders *blocksAndHeaders
	proofCache       *lru.Cache[proofRequest, *gethclient.AccountResult]
	proofRequests    *requestMap[proofRequest, *gethclient.AccountResult]
	codeCache        *lru.Cache[codeRequest, []byte]
	codeRequests     *requestMap[codeRequest, []byte]
}

func newLightState(client *rpc.Client, canonicalChain *canonicalChain, blocksAndHeaders *blocksAndHeaders) *lightState {
	s := &lightState{
		client:           client,
		canonicalChain:   canonicalChain,
		blocksAndHeaders: blocksAndHeaders,
		proofCache:       lru.NewCache[proofRequest, *gethclient.AccountResult](100),
		codeCache:        lru.NewCache[codeRequest, []byte](10),
	}
	s.proofRequests = newRequestMap[proofRequest, *gethclient.AccountResult](s.requestProof)
	s.codeRequests = newRequestMap[codeRequest, []byte](s.requestCode)
	return s
}

func (s *lightState) fetchProof(ctx context.Context, req proofRequest) (*gethclient.AccountResult, error) {
	if proof, ok := s.proofCache.Get(req); ok {
		return proof, nil
	}
	request := s.proofRequests.request(req)
	proof, err := request.getResult(ctx)
	if err == nil {
		s.proofCache.Add(req, proof) //TODO cached before validation; remove and retry if invalid
	}
	request.release()
	return proof, err
}

func (s *lightState) requestProof(ctx context.Context, req proofRequest) (*gethclient.AccountResult, error) {
	type storageResult struct {
		Key   string       `json:"key"`
		Value *hexutil.Big `json:"value"`
		Proof []string     `json:"proof"`
	}

	type accountResult struct {
		Address      common.Address  `json:"address"`
		AccountProof []string        `json:"accountProof"`
		Balance      *hexutil.Big    `json:"balance"`
		CodeHash     common.Hash     `json:"codeHash"`
		Nonce        hexutil.Uint64  `json:"nonce"`
		StorageHash  common.Hash     `json:"storageHash"`
		StorageProof []storageResult `json:"storageProof"`
	}

	var storageKeys []string
	if len(req.storageKeys) > 0 {
		storageKeys = strings.Split(req.storageKeys, ",")
	}
	log.Debug("Starting RPC request", "type", "eth_getProof", "blockNumber", req.blockNumber, "address", req.address, "storageKeys", len(storageKeys))
	var res accountResult
	err := s.client.CallContext(ctx, &res, "eth_getProof", req.address, storageKeys, hexutil.EncodeUint64(req.blockNumber))
	log.Debug("Finished RPC request", "type", "eth_getProof", "blockNumber", req.blockNumber, "address", req.address, "storageKeys", len(storageKeys), "error", err)
	var proof *gethclient.AccountResult
	if err == nil { //TODO de-duplicate
		// Turn hexutils back to normal datatypes
		storageResults := make([]gethclient.StorageResult, 0, len(res.StorageProof))
		for _, st := range res.StorageProof {
			storageResults = append(storageResults, gethclient.StorageResult{
				Key:   st.Key,
				Value: st.Value.ToInt(),
				Proof: st.Proof,
			})
		}
		proof = &gethclient.AccountResult{
			Address:      res.Address,
			AccountProof: res.AccountProof,
			Balance:      res.Balance.ToInt(),
			Nonce:        uint64(res.Nonce),
			CodeHash:     res.CodeHash,
			StorageHash:  res.StorageHash,
			StorageProof: storageResults,
		}
	}
	return proof, err
}

func (s *lightState) fetchCode(ctx context.Context, req codeRequest) ([]byte, error) {
	if code, ok := s.codeCache.Get(req); ok {
		return code, nil
	}
	request := s.codeRequests.request(req)
	code, err := request.getResult(ctx)
	if err == nil {
		s.codeCache.Add(req, code) //TODO cached before validation; remove and retry if invalid
	}
	request.release()
	return code, err
}

func (s *lightState) requestCode(ctx context.Context, req codeRequest) ([]byte, error) {
	var code hexutil.Bytes
	log.Debug("Starting RPC request", "type", "eth_getCode", "blockNumber", req.blockNumber, "address", req.address)
	err := s.client.CallContext(ctx, &code, "eth_getCode", req.address, hexutil.EncodeUint64(req.blockNumber))
	log.Debug("Finished RPC request", "type", "eth_getCode", "blockNumber", req.blockNumber, "address", req.address, "error", err)
	return code, err
}

// proofReader implements ethdb.KeyValueReader.
type proofReader map[string][]byte

func (p proofReader) Has(key []byte) (bool, error) {
	_, ok := p[string(key)]
	return ok, nil
}

func (p proofReader) Get(key []byte) ([]byte, error) {
	if value, ok := p[string(key)]; ok {
		return value, nil
	}
	return nil, errors.New("not found")
}

func makeProofReader(proof []string) (proofReader, error) {
	pr := make(proofReader)
	for _, s := range proof {
		node, err := hexutil.Decode(s)
		if err != nil {
			return nil, err
		}
		pr[string(crypto.Keccak256(node))] = node
	}
	return pr, nil
}

func stValueBytes(value *big.Int) ([]byte, error) {
	if value == nil {
		return nil, errors.New("storage value is nil")
	}
	switch value.Sign() {
	case -1:
		return nil, errors.New("negative storage value")
	case 1:
		if value.BitLen() > 256 {
			return nil, errors.New("storage value bigger than uint256")
		}
		stv := make([]byte, 32)
		value.FillBytes(stv)
		return stv, nil
	default:
		return nil, nil
	}
}

func (s *lightState) getProof(ctx context.Context, blockNumber *big.Int, account common.Address, storageKeys []string, getCode bool) (*gethclient.AccountResult, []byte, error) {
	num, pheader, err := s.canonicalChain.resolveBlockNumber(blockNumber)
	if err != nil {
		return nil, nil, err
	}
	var (
		stateRoot    common.Hash
		stateRootErr error
		stateRootCh  = make(chan struct{})
	)
	if pheader != nil {
		stateRoot = pheader.StateRoot()
		close(stateRootCh)
	} else {
		go func() {
			defer close(stateRootCh)

			blockHash, err := s.canonicalChain.getHash(ctx, num)
			if err != nil {
				stateRootErr = err
				return
			}
			if pheader := s.blocksAndHeaders.getPayloadHeader(blockHash); pheader != nil {
				stateRoot = pheader.StateRoot()
				return
			}
			header, err := s.blocksAndHeaders.getHeader(ctx, blockHash)
			if err != nil {
				stateRootErr = err
				return
			}
			stateRoot = header.Root
		}()
	}
	var (
		code    []byte
		codeErr error
		codeCh  = make(chan struct{})
	)
	if getCode {
		go func() {
			code, codeErr = s.fetchCode(ctx, codeRequest{blockNumber: num, address: account})
			close(codeCh)
		}()
	}
	proof, proofErr := s.fetchProof(ctx, proofRequest{blockNumber: num, address: account, storageKeys: strings.Join(storageKeys, ",")})
	if proofErr != nil {
		return nil, nil, proofErr
	}
	<-stateRootCh
	if stateRootErr != nil {
		return nil, nil, stateRootErr
	}
	proofReader, err := makeProofReader(proof.AccountProof)
	if err != nil {
		return nil, nil, err
	}
	value, err := trie.VerifyProof(stateRoot, crypto.Keccak256(account.Bytes()), proofReader)
	if err != nil {
		return nil, nil, err
	}
	if proof.Balance == nil {
		return nil, nil, errors.New("account balance is nil")
	}
	balance, overflow := uint256.FromBig(proof.Balance)
	if overflow {
		return nil, nil, errors.New("account balance overflow")
	}
	stateAccount := types.StateAccount{
		Nonce:    proof.Nonce,
		Balance:  balance,
		Root:     proof.StorageHash,
		CodeHash: proof.CodeHash.Bytes(),
	}
	enc, _ := rlp.EncodeToBytes(&stateAccount)
	if !bytes.Equal(enc, value) {
		return nil, nil, errors.New("account RLP mismatch")
	}
	if len(storageKeys) != len(proof.StorageProof) {
		return nil, nil, errors.New("invalid number of storage proofs")
	}
	for i, st := range proof.StorageProof {
		if proof.StorageHash == types.EmptyRootHash {
			// no storage trie, expect empty proofs and values
			if len(st.Proof) != 0 {
				return nil, nil, errors.New("non-empty storage proof from empty storage")
			}
			value, err := stValueBytes(st.Value)
			if err != nil {
				return nil, nil, err
			}
			if value != nil {
				return nil, nil, errors.New("non-empty storage value from empty storage")
			}
			continue
		}
		proofReader, err := makeProofReader(st.Proof)
		if err != nil {
			return nil, nil, err
		}
		key, err := hexutil.Decode(storageKeys[i])
		if err != nil {
			return nil, nil, err
		}
		key = common.BytesToHash(key).Bytes() // TODO 32 byte padding needed???
		value, err := trie.VerifyProof(proof.StorageHash, crypto.Keccak256(key), proofReader)
		if err != nil {
			return nil, nil, err
		}
		stv, err := stValueBytes(st.Value)
		if err != nil {
			return nil, nil, err
		}
		enc, _ := rlp.EncodeToBytes(stv)
		if !bytes.Equal(enc, value) { //TODO check for empty value
			//log.Info("storage value mismatch", "value", enc, "proven", value)
			return nil, nil, errors.New("storage value mismatch")
		}
	}
	if getCode {
		<-codeCh
		if codeErr != nil {
			return nil, nil, codeErr
		}
		if crypto.Keccak256Hash(code) != proof.CodeHash {
			return nil, nil, errors.New("code hash mismatch")
		}
	}
	return proof, code, nil
}
