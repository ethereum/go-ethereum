package types

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// DelegationPrefix is used by code to denote the account is delegating to
// another account.
var DelegationPrefix = []byte{0xef, 0x01, 0x00}

// ParseDelegation tries to parse the address from a delegation slice.
func ParseDelegation(b []byte) (common.Address, bool) {
	if len(b) != 23 || !bytes.HasPrefix(b, DelegationPrefix) {
		return common.Address{}, false
	}
	var addr common.Address
	copy(addr[:], b[len(DelegationPrefix):])
	return addr, true
}

// AddressToDelegation adds the delegation prefix to the specified address.
func AddressToDelegation(addr common.Address) []byte {
	return append(DelegationPrefix, addr.Bytes()...)
}

// SetCodeTx implements the EIP-7702 transaction type which temporarily installs
// the code at the signer's address.
type SetCodeTx struct {
	ChainID    *uint256.Int
	Nonce      uint64
	GasTipCap  *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *uint256.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         common.Address
	Value      *uint256.Int
	Data       []byte
	AccessList AccessList
	AuthList   AuthorizationList

	// Signature values
	V *uint256.Int `json:"v" gencodec:"required"`
	R *uint256.Int `json:"r" gencodec:"required"`
	S *uint256.Int `json:"s" gencodec:"required"`
}

//go:generate go run github.com/fjl/gencodec -type Authorization -field-override authorizationMarshaling -out gen_authorization.go

// Authorization is an authorization from an account to deploy code at it's
// address.
type Authorization struct {
	ChainID *big.Int       `json:"chainId" gencodec:"required"`
	Address common.Address `json:"address" gencodec:"required"`
	Nonce   uint64         `json:"nonce" gencodec:"required"`
	V       *big.Int       `json:"v" gencodec:"required"`
	R       *big.Int       `json:"r" gencodec:"required"`
	S       *big.Int       `json:"s" gencodec:"required"`
}

// field type overrides for gencodec
type authorizationMarshaling struct {
	ChainID *hexutil.Big
	Nonce   hexutil.Uint64
	V       *hexutil.Big
	R       *hexutil.Big
	S       *hexutil.Big
}

func SignAuth(auth *Authorization, prv *ecdsa.PrivateKey) (*Authorization, error) {
	h := prefixedRlpHash(
		0x05,
		[]interface{}{
			auth.ChainID,
			auth.Address,
			auth.Nonce,
		})

	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return auth.WithSignature(sig), nil
}

func (a *Authorization) WithSignature(sig []byte) *Authorization {
	r, s, _ := decodeSignature(sig)
	v := big.NewInt(int64(sig[64]))
	cpy := Authorization{
		ChainID: a.ChainID,
		Address: a.Address,
		Nonce:   a.Nonce,
		V:       v,
		R:       r,
		S:       s,
	}
	return &cpy
}

type AuthorizationList []*Authorization

func (a Authorization) Authority() (common.Address, error) {
	sighash := prefixedRlpHash(
		0x05,
		[]interface{}{
			a.ChainID,
			a.Address,
			a.Nonce,
		})

	v := byte(a.V.Uint64())
	if !crypto.ValidateSignatureValues(v, a.R, a.S, true) {
		return common.Address{}, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := a.R.Bytes(), a.S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = v
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

type SetCodeDelegation struct {
	From   common.Address
	Nonce  uint64
	Target common.Address
}

/*
func (a AuthorizationList) WithAddress() []AuthorizationTuple {
	var (
		list = make([]AuthorizationTuple, len(a))
		sha  = hasherPool.Get().(crypto.KeccakState)
		h    = common.Hash{}
	)
	defer hasherPool.Put(sha)
	for i, auth := range a {
		sha.Reset()
		sha.Write([]byte{0x05})
		sha.Write(auth.Code)
		sha.Read(h[:])
		addr, err := recoverPlain(h, auth.R, auth.S, auth.V, false)
		if err != nil {
			continue
		}
		list[i].Address = addr
		copy(list[i].Code, auth.Code)
	}
	return list
}
*/

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *SetCodeTx) copy() TxData {
	cpy := &SetCodeTx{
		Nonce: tx.Nonce,
		To:    tx.To,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		AuthList:   make(AuthorizationList, len(tx.AuthList)),
		Value:      new(uint256.Int),
		ChainID:    new(uint256.Int),
		GasTipCap:  new(uint256.Int),
		GasFeeCap:  new(uint256.Int),
		V:          new(uint256.Int),
		R:          new(uint256.Int),
		S:          new(uint256.Int),
	}
	copy(cpy.AccessList, tx.AccessList)
	copy(cpy.AuthList, tx.AuthList)
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	return cpy
}

// accessors for innerTx.
func (tx *SetCodeTx) txType() byte           { return SetCodeTxType }
func (tx *SetCodeTx) chainID() *big.Int      { return tx.ChainID.ToBig() }
func (tx *SetCodeTx) accessList() AccessList { return tx.AccessList }
func (tx *SetCodeTx) data() []byte           { return tx.Data }
func (tx *SetCodeTx) gas() uint64            { return tx.Gas }
func (tx *SetCodeTx) gasFeeCap() *big.Int    { return tx.GasFeeCap.ToBig() }
func (tx *SetCodeTx) gasTipCap() *big.Int    { return tx.GasTipCap.ToBig() }
func (tx *SetCodeTx) gasPrice() *big.Int     { return tx.GasFeeCap.ToBig() }
func (tx *SetCodeTx) value() *big.Int        { return tx.Value.ToBig() }
func (tx *SetCodeTx) nonce() uint64          { return tx.Nonce }
func (tx *SetCodeTx) to() *common.Address    { tmp := tx.To; return &tmp }

func (tx *SetCodeTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap.ToBig())
	}
	tip := dst.Sub(tx.GasFeeCap.ToBig(), baseFee)
	if tip.Cmp(tx.GasTipCap.ToBig()) > 0 {
		tip.Set(tx.GasTipCap.ToBig())
	}
	return tip.Add(tip, baseFee)
}

func (tx *SetCodeTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V.ToBig(), tx.R.ToBig(), tx.S.ToBig()
}

func (tx *SetCodeTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID.SetFromBig(chainID)
	tx.V.SetFromBig(v)
	tx.R.SetFromBig(r)
	tx.S.SetFromBig(s)
}

func (tx *SetCodeTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *SetCodeTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}
