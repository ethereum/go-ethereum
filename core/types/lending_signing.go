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

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
	"github.com/XinFinOrg/XDPoSChain/log"
)

// LendingSigner interface for lending signer transaction
type LendingSigner interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *LendingTransaction) (common.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *LendingTransaction, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *LendingTransaction) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(LendingSigner) bool
}

type lendingsigCache struct {
	signer LendingSigner
	from   common.Address
}

// LendingSender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func LendingSender(signer LendingSigner, tx *LendingTransaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(lendingsigCache)
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
	tx.from.Store(lendingsigCache{signer: signer, from: addr})
	return addr, nil
}

// LendingSignTx signs the lending transaction using the given lending signer and private key
func LendingSignTx(tx *LendingTransaction, s LendingSigner, prv *ecdsa.PrivateKey) (*LendingTransaction, error) {
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

//LendingTxSigner signer
type LendingTxSigner struct{}

// Equal compare two signer
func (lendingsign LendingTxSigner) Equal(s2 LendingSigner) bool {
	_, ok := s2.(LendingSigner)
	return ok
}

//SignatureValues returns signature values. This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (lendingsign LendingTxSigner) SignatureValues(tx *LendingTransaction, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

// LendingCreateHash hash of new lending transaction
func (lendingsign LendingTxSigner) LendingCreateHash(tx *LendingTransaction) common.Hash {
	log.Debug("LendingCreateHash", "relayer", tx.RelayerAddress().Hex(), "useraddress", tx.UserAddress().Hex(),
		"collateral", tx.CollateralToken().Hex(), "lending", tx.LendingToken().Hex(), "quantity", tx.Quantity(), "term", tx.Term(),
		"interest", tx.Interest(), "side", tx.Side, "status", tx.Status(), "type", tx.Type(), "nonce", tx.Nonce())
	borrowing := tx.Side() == LendingSideBorrow
	sha := sha3.NewKeccak256()
	sha.Write(tx.RelayerAddress().Bytes())
	sha.Write(tx.UserAddress().Bytes())
	if borrowing {
		sha.Write(tx.CollateralToken().Bytes())
	}
	sha.Write(tx.LendingToken().Bytes())
	sha.Write(common.BigToHash(tx.Quantity()).Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Term()))).Bytes())
	if tx.IsLoTypeLending() {
		sha.Write(common.BigToHash(big.NewInt(int64(tx.Interest()))).Bytes())
	}
	sha.Write([]byte(tx.Side()))
	sha.Write([]byte(tx.Status()))
	sha.Write([]byte(tx.Type()))
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	if borrowing {
		autoTopUp := int64(0)
		if tx.AutoTopUp() {
			autoTopUp = int64(1)
		}
		sha.Write(common.BigToHash(big.NewInt(autoTopUp)).Bytes())
	}
	return common.BytesToHash(sha.Sum(nil))
}

// LendingCancelHash hash of cancelled lending transaction
func (lendingsign LendingTxSigner) LendingCancelHash(tx *LendingTransaction) common.Hash {
	sha := sha3.NewKeccak256()
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	sha.Write([]byte(tx.Status()))
	sha.Write(tx.RelayerAddress().Bytes())
	sha.Write(tx.UserAddress().Bytes())
	sha.Write(tx.LendingToken().Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Term()))).Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.LendingId()))).Bytes())
	return common.BytesToHash(sha.Sum(nil))
}

// LendingRepayHash hash of cancelled lending transaction
func (lendingsign LendingTxSigner) LendingRepayHash(tx *LendingTransaction) common.Hash {
	sha := sha3.NewKeccak256()
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	sha.Write([]byte(tx.Status()))
	sha.Write(tx.RelayerAddress().Bytes())
	sha.Write(tx.UserAddress().Bytes())
	sha.Write(tx.LendingToken().Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Term()))).Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.LendingTradeId()))).Bytes())
	sha.Write([]byte(tx.Type()))
	return common.BytesToHash(sha.Sum(nil))
}

// LendingTopUpHash hash of cancelled lending transaction
func (lendingsign LendingTxSigner) LendingTopUpHash(tx *LendingTransaction) common.Hash {
	sha := sha3.NewKeccak256()
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	sha.Write([]byte(tx.Status()))
	sha.Write(tx.RelayerAddress().Bytes())
	sha.Write(tx.UserAddress().Bytes())
	sha.Write(tx.LendingToken().Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Term()))).Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.LendingTradeId()))).Bytes())
	sha.Write(common.BigToHash(tx.Quantity()).Bytes())
	sha.Write([]byte(tx.Type()))
	return common.BytesToHash(sha.Sum(nil))
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (lendingsign LendingTxSigner) Hash(tx *LendingTransaction) common.Hash {
	if tx.IsCancelledLending() {
		return lendingsign.LendingCancelHash(tx)
	}
	if tx.IsCreatedLending() {
		return lendingsign.LendingCreateHash(tx)
	}
	if tx.IsTopupLending() {
		return lendingsign.LendingTopUpHash(tx)
	}
	if tx.IsRepayLending() {
		return lendingsign.LendingRepayHash(tx)
	}
	return common.Hash{}
}

// Sender get signer from
func (lendingsign LendingTxSigner) Sender(tx *LendingTransaction) (common.Address, error) {

	message := crypto.Keccak256(
		[]byte("\x19Ethereum Signed Message:\n32"),
		lendingsign.Hash(tx).Bytes(),
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

// CacheLendingSigner cache signed lending transaction
func CacheLendingSigner(signer LendingSigner, tx *LendingTransaction) {
	if tx == nil {
		return
	}
	addr, err := signer.Sender(tx)
	if err != nil {
		return
	}
	tx.from.Store(lendingsigCache{signer: signer, from: addr})
}
