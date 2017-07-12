// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"

	"errors"

	"context"

	"math/big"

	"io/ioutil"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"encoding/json"
)

type SignerAPI struct {
	chainID *big.Int
	am      *accounts.Manager
}

// NewSignerAPI creates a new API that can be used for account management.
// ksLocation specifies the directory where to store the password protected private
// key that is generated when a new account is created.
// noUSB disables USB support that is required to support hardware devices such as
// ledger and trezor.
func NewSignerAPI(chainID int64, ksLocation string, noUSB bool) *SignerAPI {
	var backends []accounts.Backend

	// support password based accounts
	if len(ksLocation) > 0 {
		backends = append(backends, keystore.NewKeyStore(ksLocation, keystore.StandardScryptN, keystore.StandardScryptP))
	}

	if !noUSB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			backends = append(backends, ledgerhub)
			log.Debug("Ledger support enabled")
		}
		// Start a USB hub for Trezor hardware wallets
		if trezorhub, err := usbwallet.NewTrezorHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Trezor hub, disabling: %v", err))
		} else {
			backends = append(backends, trezorhub)
			log.Debug("Trezor support enabled")
		}
	}

	return &SignerAPI{big.NewInt(chainID), accounts.NewManager(backends...)}
}

// List returns the set of wallet this signer manages. Each wallet can contain
// multiple accounts.
func (api *SignerAPI) List(ctx context.Context) []Account {
	var accounts []Account
	for _, wallet := range api.am.Wallets() {
		for _, acc := range wallet.Accounts() {
			acc := Account{Typ: "account", URL: wallet.URL(), Address: acc.Address}
			accounts = append(accounts, acc)
		}
	}
	return accounts
}

// New creates a new password protected account. The private key is protected with
// the given password. Users are responsible to backup the private key that is stored
// in the keystore location thas was specified when this API was created.
func (api *SignerAPI) New(ctx context.Context, passphrase string) (accounts.Account, error) {
	be := api.am.Backends(keystore.KeyStoreType)
	if len(be) == 0 {
		return accounts.Account{}, errors.New("password based accounts not supported")
	}
	acc, err := be[0].(*keystore.KeyStore).NewAccount(passphrase)
	return acc, err
}

// SignTransaction signs the given transaction and returns it in an RLP encoded form
// that can be posted to `eth_sendRawTransaction`.
func (api *SignerAPI) SignTransaction(ctx context.Context, from common.Address, passwd string, args TransactionArg) (hexutil.Bytes, error) {
	acc := accounts.Account{Address: from}

	wallet, err := api.am.Find(acc)
	if err != nil {
		return nil, err
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), (*big.Int)(args.Gas), (*big.Int)(args.GasPrice), args.Data)
	} else {
		tx = types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), (*big.Int)(args.Gas), (*big.Int)(args.GasPrice), args.Data)
	}

	signedTx, err := wallet.SignTxWithPassphrase(acc, passwd, tx, api.chainID)
	if err != nil {
		return nil, err
	}

	return rlp.EncodeToBytes(signedTx)
}

// Sign calculates an Ethereum ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message))
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The key used to calculate the signature is decrypted with the given password.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_sign
func (api *SignerAPI) Sign(ctx context.Context, addr common.Address, passwd string, data hexutil.Bytes) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := api.am.Find(account)
	if err != nil {
		return nil, err
	}
	// Assemble sign the data with the wallet
	signature, err := wallet.SignHashWithPassphrase(account, passwd, signHash(data))
	if err != nil {
		return nil, err
	}
	signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return signature, nil
}

// EcRecover returns the address for the account that was used to create the signature.
// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
// the address of:
// hash = keccak256("\x19Ethereum Signed Message:\n"${message length}${message})
// addr = ecrecover(hash, signature)
//
// Note, the signature must conform to the secp256k1 curve R, S and V values, where
// the V value must be be 27 or 28 for legacy reasons.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_ecRecover
func (api *SignerAPI) EcRecover(ctx context.Context, data, sig hexutil.Bytes) (common.Address, error) {
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("signature must be 65 bytes long")
	}
	if sig[64] != 27 && sig[64] != 28 {
		return common.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1

	rpk, err := crypto.Ecrecover(signHash(data), sig)
	if err != nil {
		return common.Address{}, err
	}
	pubKey := crypto.ToECDSAPub(rpk)
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddr, nil
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

// Export returns encrypted private key associated with the given address in web3 keystore format.
func (api *SignerAPI) Export(ctx context.Context, addr common.Address) (json.RawMessage, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := api.am.Find(account)
	if err != nil {
		return nil, err
	}

	url := wallet.URL()
	if url.Scheme != keystore.KeyStoreScheme {
		return nil, fmt.Errorf("account is not a password protected account")
	}

	return ioutil.ReadFile(url.Path)
}

// Imports tries to import the given keyJSON in the local keystore. The keyJSON data is expected to be
// in web3 keystore format. It will decrypt the keyJSON with the given passphrase and on successful
// decryption it will encrypt the key with the given newPassphrase and store it in the keystore.
func (api *SignerAPI) Import(ctx context.Context, keyJSON json.RawMessage, passphrase, newPassphrase string) (Account, error) {
	be := api.am.Backends(keystore.KeyStoreType)
	if len(be) == 0 {
		return Account{}, errors.New("password based accounts not supported")
	}

	acc, err := be[0].(*keystore.KeyStore).Import(keyJSON, passphrase, newPassphrase)
	if err != nil {
		return Account{}, err
	}

	return Account{Typ: "account", URL: acc.URL, Address: acc.Address}, nil
}