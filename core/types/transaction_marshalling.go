// Copyright 2021 The go-ethereum Authors
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

package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/protolambda/ztyp/view"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// txJSON is the JSON representation of transactions.
type txJSON struct {
	Type hexutil.Uint64 `json:"type"`

	// Common transaction fields:
	Nonce                *hexutil.Uint64 `json:"nonce"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	Value                *hexutil.Big    `json:"value"`
	Data                 *hexutil.Bytes  `json:"input"`
	V                    *hexutil.Big    `json:"v"`
	R                    *hexutil.Big    `json:"r"`
	S                    *hexutil.Big    `json:"s"`
	To                   *common.Address `json:"to"`

	// Access list transaction fields:
	ChainID    *hexutil.Big `json:"chainId,omitempty"`
	AccessList *AccessList  `json:"accessList,omitempty"`

	// Blob transaction fields:
	MaxFeePerDataGas    *hexutil.Big  `json:"maxFeePerDataGas,omitempty"`
	BlobVersionedHashes []common.Hash `json:"blobVersionedHashes,omitempty"`
	Blobs               Blobs         `json:"blobs,omitempty"`
	BlobKzgs            BlobKzgs      `json:"blobKzgs,omitempty"`
	KzgAggregatedProof  KZGProof      `json:"kzgAggregatedProof,omitempty"`

	// Only used for encoding:
	Hash common.Hash `json:"hash"`
}

// MarshalJSON marshals as JSON with a hash.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	var enc txJSON
	// These are set for all tx types.
	enc.Hash = tx.Hash()
	enc.Type = hexutil.Uint64(tx.Type())

	// Other fields are set conditionally depending on tx type.
	switch itx := tx.inner.(type) {
	case *LegacyTx:
		enc.Nonce = (*hexutil.Uint64)(&itx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&itx.Gas)
		enc.GasPrice = (*hexutil.Big)(itx.GasPrice)
		enc.Value = (*hexutil.Big)(itx.Value)
		enc.Data = (*hexutil.Bytes)(&itx.Data)
		enc.To = tx.To()
		enc.V = (*hexutil.Big)(itx.V)
		enc.R = (*hexutil.Big)(itx.R)
		enc.S = (*hexutil.Big)(itx.S)
	case *AccessListTx:
		enc.ChainID = (*hexutil.Big)(itx.ChainID)
		enc.AccessList = &itx.AccessList
		enc.Nonce = (*hexutil.Uint64)(&itx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&itx.Gas)
		enc.GasPrice = (*hexutil.Big)(itx.GasPrice)
		enc.Value = (*hexutil.Big)(itx.Value)
		enc.Data = (*hexutil.Bytes)(&itx.Data)
		enc.To = tx.To()
		enc.V = (*hexutil.Big)(itx.V)
		enc.R = (*hexutil.Big)(itx.R)
		enc.S = (*hexutil.Big)(itx.S)
	case *DynamicFeeTx:
		enc.ChainID = (*hexutil.Big)(itx.ChainID)
		enc.AccessList = &itx.AccessList
		enc.Nonce = (*hexutil.Uint64)(&itx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&itx.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(itx.GasFeeCap)
		enc.MaxPriorityFeePerGas = (*hexutil.Big)(itx.GasTipCap)
		enc.Value = (*hexutil.Big)(itx.Value)
		enc.Data = (*hexutil.Bytes)(&itx.Data)
		enc.To = tx.To()
		enc.V = (*hexutil.Big)(itx.V)
		enc.R = (*hexutil.Big)(itx.R)
		enc.S = (*hexutil.Big)(itx.S)
	case *SignedBlobTx:
		enc.ChainID = (*hexutil.Big)(u256ToBig(&itx.Message.ChainID))
		enc.AccessList = (*AccessList)(&itx.Message.AccessList)
		enc.Nonce = (*hexutil.Uint64)(&itx.Message.Nonce)
		enc.Gas = (*hexutil.Uint64)(&itx.Message.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(u256ToBig(&itx.Message.GasFeeCap))
		enc.MaxPriorityFeePerGas = (*hexutil.Big)(u256ToBig(&itx.Message.GasTipCap))
		enc.Value = (*hexutil.Big)(u256ToBig(&itx.Message.Value))
		enc.Data = (*hexutil.Bytes)(&itx.Message.Data)
		enc.To = tx.To()
		v, r, s := tx.RawSignatureValues()
		enc.V = (*hexutil.Big)(v)
		enc.R = (*hexutil.Big)(r)
		enc.S = (*hexutil.Big)(s)
		enc.MaxFeePerDataGas = (*hexutil.Big)(u256ToBig(&itx.Message.MaxFeePerDataGas))
		enc.BlobVersionedHashes = itx.Message.BlobVersionedHashes
		if tx.wrapData != nil {
			enc.Blobs = tx.wrapData.blobs()
			enc.BlobKzgs = tx.wrapData.kzgs()
			enc.KzgAggregatedProof = tx.wrapData.aggregatedProof()
		}
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}

	// Decode / verify fields according to transaction type.
	var inner TxData
	switch dec.Type {
	case LegacyTxType:
		var itx LegacyTx
		inner = &itx
		if dec.To != nil {
			itx.To = dec.To
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Nonce = uint64(*dec.Nonce)
		if dec.GasPrice == nil {
			return errors.New("missing required field 'gasPrice' in transaction")
		}
		itx.GasPrice = (*big.Int)(dec.GasPrice)
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in transaction")
		}
		itx.Gas = uint64(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Value = (*big.Int)(dec.Value)
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Data = *dec.Data
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.S = (*big.Int)(dec.S)
		withSignature := itx.V.Sign() != 0 || itx.R.Sign() != 0 || itx.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(itx.V, itx.R, itx.S, true); err != nil {
				return err
			}
		}

	case AccessListTxType:
		var itx AccessListTx
		inner = &itx
		// Access list is optional for now.
		if dec.AccessList != nil {
			itx.AccessList = *dec.AccessList
		}
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		itx.ChainID = (*big.Int)(dec.ChainID)
		if dec.To != nil {
			itx.To = dec.To
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Nonce = uint64(*dec.Nonce)
		if dec.GasPrice == nil {
			return errors.New("missing required field 'gasPrice' in transaction")
		}
		itx.GasPrice = (*big.Int)(dec.GasPrice)
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in transaction")
		}
		itx.Gas = uint64(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Value = (*big.Int)(dec.Value)
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Data = *dec.Data
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.S = (*big.Int)(dec.S)
		withSignature := itx.V.Sign() != 0 || itx.R.Sign() != 0 || itx.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(itx.V, itx.R, itx.S, false); err != nil {
				return err
			}
		}

	case DynamicFeeTxType:
		var itx DynamicFeeTx
		inner = &itx
		// Access list is optional for now.
		if dec.AccessList != nil {
			itx.AccessList = *dec.AccessList
		}
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		itx.ChainID = (*big.Int)(dec.ChainID)
		if dec.To != nil {
			itx.To = dec.To
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Nonce = uint64(*dec.Nonce)
		if dec.MaxPriorityFeePerGas == nil {
			return errors.New("missing required field 'maxPriorityFeePerGas' for txdata")
		}
		itx.GasTipCap = (*big.Int)(dec.MaxPriorityFeePerGas)
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		itx.GasFeeCap = (*big.Int)(dec.MaxFeePerGas)
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' for txdata")
		}
		itx.Gas = uint64(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Value = (*big.Int)(dec.Value)
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Data = *dec.Data
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.S = (*big.Int)(dec.S)
		withSignature := itx.V.Sign() != 0 || itx.R.Sign() != 0 || itx.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(itx.V, itx.R, itx.S, false); err != nil {
				return err
			}
		}
	case BlobTxType:
		var itx SignedBlobTx
		inner = &itx
		// Access list is optional for now.
		if dec.AccessList != nil {
			itx.Message.AccessList = AccessListView(*dec.AccessList)
		}
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		itx.Message.ChainID.SetFromBig((*big.Int)(dec.ChainID))
		if dec.To != nil {
			itx.Message.To.Address = (*AddressSSZ)(dec.To)
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Message.Nonce = view.Uint64View(*dec.Nonce)
		if dec.MaxPriorityFeePerGas == nil {
			return errors.New("missing required field 'maxPriorityFeePerGas' for txdata")
		}
		itx.Message.GasTipCap.SetFromBig((*big.Int)(dec.MaxPriorityFeePerGas))
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		itx.Message.GasFeeCap.SetFromBig((*big.Int)(dec.MaxFeePerGas))
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' for txdata")
		}
		itx.Message.Gas = view.Uint64View(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Message.Value.SetFromBig((*big.Int)(dec.Value))
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Message.Data = TxDataView(*dec.Data)
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.Signature.V = view.Uint8View((*big.Int)(dec.V).Uint64())
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.Signature.R.SetFromBig((*big.Int)(dec.R))
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.Signature.S.SetFromBig((*big.Int)(dec.S))
		withSignature := (*big.Int)(dec.V).Sign() != 0 || (*big.Int)(dec.R).Sign() != 0 || (*big.Int)(dec.S).Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(big.NewInt(int64(itx.Signature.V)), u256ToBig(&itx.Signature.R), u256ToBig(&itx.Signature.S), false); err != nil {
				return err
			}
		}
		itx.Message.MaxFeePerDataGas.SetFromBig((*big.Int)(dec.MaxFeePerDataGas))
		if dec.MaxFeePerDataGas == nil {
			return errors.New("missing required field 'maxFeePerDataGas' for txdata")
		}
		itx.Message.BlobVersionedHashes = dec.BlobVersionedHashes
		// A BlobTx may not contain data
		if len(dec.Blobs) != 0 || len(dec.BlobKzgs) != 0 {
			tx.wrapData = &BlobTxWrapData{
				BlobKzgs:           dec.BlobKzgs,
				Blobs:              dec.Blobs,
				KzgAggregatedProof: dec.KzgAggregatedProof,
			}
			// Verify that versioned hashes match kzgs, and kzgs match blobs.
			if err := tx.wrapData.validateBlobTransactionWrapper(&itx); err != nil {
				return fmt.Errorf("blob wrapping data is invalid: %v", err)
			}
		}
	default:
		return ErrTxTypeNotSupported
	}

	// Now set the inner transaction.
	tx.setDecoded(inner, 0)

	// TODO: check hash here?
	return nil
}
