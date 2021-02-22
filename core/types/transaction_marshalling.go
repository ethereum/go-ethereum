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
	AccountNonce *hexutil.Uint64 `json:"nonce"`
	Price        *hexutil.Big    `json:"gasPrice"`
	GasLimit     *hexutil.Uint64 `json:"gas"`
	Amount       *hexutil.Big    `json:"value"`
	Payload      *hexutil.Bytes  `json:"input"`
	V            *hexutil.Big    `json:"v"`
	R            *hexutil.Big    `json:"r"`
	S            *hexutil.Big    `json:"s"`
	Recipient    *common.Address `json:"to"`

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
		enc.AccountNonce = (*hexutil.Uint64)(&tx.AccountNonce)
		enc.GasLimit = (*hexutil.Uint64)(&tx.GasLimit)
		enc.Price = (*hexutil.Big)(tx.Price)
		enc.Amount = (*hexutil.Big)(tx.Amount)
		enc.Payload = (*hexutil.Bytes)(&tx.Payload)
		enc.Recipient = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
	case *AccessListTx:
		enc.ChainID = (*hexutil.Big)(tx.Chain)
		enc.AccessList = tx.Accesses
		enc.AccountNonce = (*hexutil.Uint64)(&tx.AccountNonce)
		enc.GasLimit = (*hexutil.Uint64)(&tx.GasLimit)
		enc.Price = (*hexutil.Big)(tx.Price)
		enc.Amount = (*hexutil.Big)(tx.Amount)
		enc.Payload = (*hexutil.Bytes)(&tx.Payload)
		enc.Recipient = t.To()
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
		if dec.Recipient != nil {
			itx.Recipient = dec.Recipient
		}
		if dec.AccountNonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.AccountNonce = uint64(*dec.AccountNonce)
		if dec.Price == nil {
			return errors.New("missing required field 'gasPrice' in transaction")
		}
		itx.Price = (*big.Int)(dec.Price)
		if dec.GasLimit == nil {
			return errors.New("missing required field 'gas' in transaction")
		}
		itx.GasLimit = uint64(*dec.GasLimit)
		if dec.Amount == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Amount = (*big.Int)(dec.Amount)
		if dec.Payload == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Payload = *dec.Payload
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
			itx.Accesses = dec.AccessList
		}
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		itx.Chain = (*big.Int)(dec.ChainID)
		if dec.Recipient != nil {
			itx.Recipient = dec.Recipient
		}
		if dec.AccountNonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.AccountNonce = uint64(*dec.AccountNonce)
		if dec.Price == nil {
			return errors.New("missing required field 'gasPrice' in transaction")
		}
		itx.Price = (*big.Int)(dec.Price)
		if dec.GasLimit == nil {
			return errors.New("missing required field 'gas' in transaction")
		}
		itx.GasLimit = uint64(*dec.GasLimit)
		if dec.Amount == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Amount = (*big.Int)(dec.Amount)
		if dec.Payload == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Payload = *dec.Payload
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
