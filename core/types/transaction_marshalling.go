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
	"math/big"

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

	// Only used for encoding:
	Hash common.Hash `json:"hash"`
}

// MarshalJSON marshals as JSON with a hash.
func (t *Transaction) MarshalJSON() ([]byte, error) {
	var enc txJSON
	// These are set for all tx types.
	enc.Hash = t.Hash()
	enc.Type = hexutil.Uint64(t.Type())

	// Other fields are set conditionally depending on tx type.
	switch tx := t.inner.(type) {
	case *LegacyTx:
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.GasPrice = (*hexutil.Big)(tx.GasPrice)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
	case *AccessListTx:
		enc.ChainID = (*hexutil.Big)(tx.ChainID)
		enc.AccessList = &tx.AccessList
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.GasPrice = (*hexutil.Big)(tx.GasPrice)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
	case *DynamicFeeTx:
		enc.ChainID = (*hexutil.Big)(tx.ChainID)
		enc.AccessList = &tx.AccessList
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(tx.GasFeeCap)
		enc.MaxPriorityFeePerGas = (*hexutil.Big)(tx.GasTipCap)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *Transaction) UnmarshalJSON(input []byte) error {
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

	default:
		return ErrTxTypeNotSupported
	}

	// Now set the inner transaction.
	t.setDecoded(inner, 0)

	// TODO: check hash here?
	return nil
}
