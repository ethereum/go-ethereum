// Copyright 2016 The go-ethereum Authors
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
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strconv"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
)

// OrderSigner interface for order transaction
type OrderSigner interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *OrderTransaction) (common.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *OrderTransaction, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *OrderTransaction) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(OrderSigner) bool
}

type ordersigCache struct {
	signer OrderSigner
	from   common.Address
}

// OrderSender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func OrderSender(signer OrderSigner, tx *OrderTransaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(ordersigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(ordersigCache{signer: signer, from: addr})
	return addr, nil
}

// OrderSignTx signs the order transaction using the given order signer and private key
func OrderSignTx(tx *OrderTransaction, s OrderSigner, prv *ecdsa.PrivateKey) (*OrderTransaction, error) {
	h := s.Hash(tx)
	message := crypto.Keccak256(
		[]byte("\x19Ethereum Signed Message:\n32"),
		h.Bytes(),
	)

	sig, err := crypto.Sign(message[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

//OrderTxSigner signer
type OrderTxSigner struct{}

// Equal compare two signer
func (ordersign OrderTxSigner) Equal(s2 OrderSigner) bool {
	_, ok := s2.(OrderSigner)
	return ok
}

//SignatureValues returns signature values. This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (ordersign OrderTxSigner) SignatureValues(tx *OrderTransaction, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

// OrderCreateHash hash of new order
func (ordersign OrderTxSigner) OrderCreateHash(tx *OrderTransaction) common.Hash {
	sha := sha3.NewKeccak256()
	sha.Write(tx.ExchangeAddress().Bytes())
	sha.Write(tx.UserAddress().Bytes())
	sha.Write(tx.BaseToken().Bytes())
	sha.Write(tx.QuoteToken().Bytes())
	sha.Write(common.BigToHash(tx.Quantity()).Bytes())
	if tx.IsLoTypeOrder() {
		if tx.Price() != nil {
			sha.Write(common.BigToHash(tx.Price()).Bytes())
		}
	}
	sha.Write(common.BigToHash(tx.EncodedSide()).Bytes())
	sha.Write([]byte(tx.Status()))
	sha.Write([]byte(tx.Type()))
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	return common.BytesToHash(sha.Sum(nil))
}

// OrderCancelHash hash of cancelled order
func (ordersign OrderTxSigner) OrderCancelHash(tx *OrderTransaction) common.Hash {
	sha := sha3.NewKeccak256()
	sha.Write(tx.OrderHash().Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	sha.Write(tx.UserAddress().Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.OrderID()))).Bytes())
	sha.Write([]byte(tx.Status()))
	sha.Write(tx.ExchangeAddress().Bytes())
	sha.Write(tx.BaseToken().Bytes())
	sha.Write(tx.QuoteToken().Bytes())

	return common.BytesToHash(sha.Sum(nil))
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (ordersign OrderTxSigner) Hash(tx *OrderTransaction) common.Hash {
	if tx.IsCancelledOrder() {
		return ordersign.OrderCancelHash(tx)
	}
	return ordersign.OrderCreateHash(tx)
}

//MarshalSignature encode signature
func MarshalSignature(R, S, V *big.Int) ([]byte, error) {
	sigBytes1 := common.BigToHash(R).Bytes()
	sigBytes2 := common.BigToHash(S).Bytes()
	v := big.NewInt(0)
	v.Sub(V, big.NewInt(27))

	bigstr := v.String()
	n, err := strconv.ParseInt(bigstr, 10, 8)
	if err != nil {
		return nil, err
	}
	sigBytes3 := byte(n)
	sigBytes := append([]byte{}, sigBytes1...)
	sigBytes = append(sigBytes, sigBytes2...)
	sigBytes = append(sigBytes, sigBytes3)

	return sigBytes, nil
}

// Sender get signer from
func (ordersign OrderTxSigner) Sender(tx *OrderTransaction) (common.Address, error) {

	message := crypto.Keccak256(
		[]byte("\x19Ethereum Signed Message:\n32"),
		ordersign.Hash(tx).Bytes(),
	)
	V, R, S := tx.Signature()

	sigBytes, err := MarshalSignature(R, S, V)
	if err != nil {
		return common.Address{}, err
	}
	pubKey, err := crypto.SigToPub(message, sigBytes)
	if err != nil {
		return common.Address{}, err
	}
	address := crypto.PubkeyToAddress(*pubKey)
	return address, nil

}

// CacheOrderSigner cache signed order
func CacheOrderSigner(signer OrderSigner, tx *OrderTransaction) {
	if tx == nil {
		return
	}
	addr, err := signer.Sender(tx)
	if err != nil {
		return
	}
	tx.from.Store(ordersigCache{signer: signer, from: addr})
}
