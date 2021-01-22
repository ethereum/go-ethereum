package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// MarshalJSON marshals as JSON with a hash.
func (t *Transaction) MarshalJSON() ([]byte, error) {
	type txdata struct {
		Type         hexutil.Uint64  `json:"type"     rlp:"-"`
		Chain        *hexutil.Big    `json:"chainId"  rlp:"-"`
		AccountNonce hexutil.Uint64  `json:"nonce"    gencodec:"required"`
		Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		GasLimit     hexutil.Uint64  `json:"gas"      gencodec:"required"`
		Recipient    *common.Address `json:"to"       rlp:"nil"`
		Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload      hexutil.Bytes   `json:"input"    gencodec:"required"`
		AccessList   *AccessList     `json:"accessList" rlp:"-"`
		V            *hexutil.Big    `json:"v" gencodec:"required"`
		R            *hexutil.Big    `json:"r" gencodec:"required"`
		S            *hexutil.Big    `json:"s" gencodec:"required"`
		Hash         *common.Hash    `json:"hash" rlp:"-"`
	}
	var enc txdata
	enc.AccountNonce = hexutil.Uint64(t.Nonce())
	enc.Price = (*hexutil.Big)(t.GasPrice())
	enc.GasLimit = hexutil.Uint64(t.Gas())
	enc.Recipient = t.To()
	enc.Amount = (*hexutil.Big)(t.Value())
	enc.Payload = t.Data()
	v, r, s := t.RawSignatureValues()
	enc.V = (*hexutil.Big)(v)
	enc.R = (*hexutil.Big)(r)
	enc.S = (*hexutil.Big)(s)
	hash := t.Hash()
	enc.Hash = &hash
	if t.Type() == AccessListTxId {
		enc.Type = hexutil.Uint64(t.Type())
		enc.Chain = (*hexutil.Big)(t.ChainId())
		enc.AccessList = t.AccessList()
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *Transaction) UnmarshalJSON(input []byte) error {
	type txdata struct {
		Type         *hexutil.Uint64 `json:"type" rlp:"-"`
		Chain        *hexutil.Big    `json:"chainId"  rlp:"-"`
		AccountNonce *hexutil.Uint64 `json:"nonce"    gencodec:"required"`
		Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		GasLimit     *hexutil.Uint64 `json:"gas"      gencodec:"required"`
		Recipient    *common.Address `json:"to"       rlp:"nil"`
		Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload      *hexutil.Bytes  `json:"input"    gencodec:"required"`
		AccessList   *AccessList     `json:"accessList" rlp:"-"`
		V            *hexutil.Big    `json:"v" gencodec:"required"`
		R            *hexutil.Big    `json:"r" gencodec:"required"`
		S            *hexutil.Big    `json:"s" gencodec:"required"`
		Hash         *common.Hash    `json:"hash" rlp:"-"`
	}
	var dec txdata

	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.AccountNonce == nil {
		return errors.New("missing required field 'nonce' for txdata")
	}

	if dec.Type == nil || *dec.Type == hexutil.Uint64(LegacyTxId) {
		var i LegacyTransaction
		if dec.AccountNonce == nil {
			return errors.New("missing required field 'nonce' for txdata")
		}
		i.AccountNonce = uint64(*dec.AccountNonce)
		if dec.Price == nil {
			return errors.New("missing required field 'gasPrice' for txdata")
		}
		i.Price = (*big.Int)(dec.Price)
		if dec.GasLimit == nil {
			return errors.New("missing required field 'gas' for txdata")
		}
		i.GasLimit = uint64(*dec.GasLimit)
		if dec.Recipient != nil {
			i.Recipient = dec.Recipient
		}
		if dec.Amount == nil {
			return errors.New("missing required field 'value' for txdata")
		}
		i.Amount = (*big.Int)(dec.Amount)
		if dec.Payload == nil {
			return errors.New("missing required field 'input' for txdata")
		}
		i.Payload = *dec.Payload
		if dec.V == nil {
			return errors.New("missing required field 'v' for txdata")
		}
		i.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' for txdata")
		}
		i.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' for txdata")
		}
		i.S = (*big.Int)(dec.S)
		if dec.Hash != nil {
			t.hash.Store(*dec.Hash)
		}
		withSignature := i.V.Sign() != 0 || i.R.Sign() != 0 || i.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(i.V, i.R, i.S, true); err != nil {
				return err
			}
		}
		t.inner = &i
	} else if *dec.Type == hexutil.Uint64(AccessListTxId) {
		t.typ = AccessListTxId
		var i AccessListTransaction
		if dec.Chain == nil {
			return errors.New("missing required field 'chainId' for txdata")
		}
		i.Chain = (*big.Int)(dec.Chain)
		if dec.AccountNonce == nil {
			return errors.New("missing required field 'nonce' for txdata")
		}
		i.AccountNonce = uint64(*dec.AccountNonce)
		if dec.Price == nil {
			return errors.New("missing required field 'gasPrice' for txdata")
		}
		i.Price = (*big.Int)(dec.Price)
		if dec.GasLimit == nil {
			return errors.New("missing required field 'gas' for txdata")
		}
		i.GasLimit = uint64(*dec.GasLimit)
		if dec.Recipient != nil {
			i.Recipient = dec.Recipient
		}
		if dec.Amount == nil {
			return errors.New("missing required field 'value' for txdata")
		}
		i.Amount = (*big.Int)(dec.Amount)
		if dec.Payload == nil {
			return errors.New("missing required field 'input' for txdata")
		}
		i.Payload = *dec.Payload
		if dec.AccessList == nil {
			return errors.New("missing required field 'accessList' for txdata")
		}
		i.Accesses = dec.AccessList
		if dec.V == nil {
			return errors.New("missing required field 'v' for txdata")
		}
		i.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' for txdata")
		}
		i.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' for txdata")
		}
		i.S = (*big.Int)(dec.S)
		if dec.Hash != nil {
			t.hash.Store(*dec.Hash)
		}
		withSignature := i.V.Sign() != 0 || i.R.Sign() != 0 || i.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(i.V, i.R, i.S, false); err != nil {
				return err
			}
		}
		t.inner = &i
	}

	return nil
}
