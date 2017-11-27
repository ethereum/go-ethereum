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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

type SignerAPI struct {
	chainID *big.Int
	am      *accounts.Manager
	ui      SignerUI
}

// Metadata about the request
type Metadata struct {
	remote string
	local  string
	scheme string
}

// types for the requests/response types
type (
	// SignTxRequest contains info about a transaction to sign
	SignTxRequest struct {
		transaction *types.Transaction
		from        accounts.Account
		callinfo    fmt.Stringer
	}
	// SignTxResponse result from SignTxRequest
	SignTxResponse struct {
		hash     common.Hash
		approved bool
		pw       string
	}
	// ExportRequest info about query to export accounts
	ExportRequest struct {
		account accounts.Account
		file    string
	}
	// ExportResponse response to export-request
	ExportResponse struct {
		approved bool
	}
	// ImportRequest info about request to import an account
	ImportRequest struct {
		account accounts.Account
	}
	ImportResponse struct {
		approved    bool
		oldPassword string
		newPassword string
	}
	SignDataRequest struct {
		account accounts.Account
		rawdata hexutil.Bytes
		message string
		hash    hexutil.Bytes
	}
	SignDataResponse struct {
		approved bool
		pw       string
	}
	NewAccountRequest  struct{}
	NewAccountResponse struct {
		approved bool
		pw       string
	}
	ListRequest struct {
		accounts []Account
	}
	ListResponse struct {
		accounts []Account
	}
)

type errorWrapper struct {
	msg string
	err error
}

func (ew errorWrapper) String() string {
	return fmt.Sprintf("%s\n%s", ew.msg, ew.err)
}

// SignerUI specifies what method a UI needs to implement to be able to be used as a UI
// for the signer
type SignerUI interface {

	// ApproveTx prompt the user for confirmation to request to sign transaction
	ApproveTx(request *SignTxRequest, metadata Metadata, ch chan SignTxResponse)
	// ApproveSignData prompt the user for confirmation to request to sign data
	ApproveSignData(request *SignDataRequest, metadata Metadata, ch chan SignDataResponse)
	// ApproveExport prompt the user for confirmation to export encrypted account json
	ApproveExport(request *ExportRequest, metadata Metadata, ch chan ExportResponse)
	// ApproveImport prompt the user for confirmation to import account json
	ApproveImport(request *ImportRequest, metadata Metadata, ch chan ImportResponse)
	// ApproveListing prompt the user for confirmation to list accounts
	// the list of accounts to list can be modified by the ui
	ApproveListing(request *ListRequest, metadata Metadata, ch chan ListResponse)
	// ApproveNewAccount prompt the user for confirmation to create new account, and reveal to caller
	ApproveNewAccount(requst *NewAccountRequest, metadata Metadata, ch chan NewAccountResponse)
	// ShowError displays error message to user
	ShowError(message string)
	// ShowInfo displays info message to user
	ShowInfo(message string)
}

type HeadlessUI struct {
}

func (ui *HeadlessUI) ApproveTx(request *SignTxRequest, metadata Metadata, ch chan SignTxResponse) {
	ch <- SignTxResponse{request.transaction.Hash(), true, ""}
}
func (ui *HeadlessUI) ApproveSignData(request *SignDataRequest, metadata Metadata, ch chan SignDataResponse) {
	ch <- SignDataResponse{true, ""}
}
func (ui *HeadlessUI) ApproveExport(request *ExportRequest, metadata Metadata, ch chan ExportResponse) {
	ch <- ExportResponse{true}
}
func (ui *HeadlessUI) ApproveImport(request *ImportRequest, metadata Metadata, ch chan ImportResponse) {
	ch <- ImportResponse{true, "", ""}
}
func (ui *HeadlessUI) ApproveListing(request *ListRequest, metadata Metadata, ch chan ListResponse) {
	ch <- ListResponse{request.accounts}
}
func (ui *HeadlessUI) ApproveNewAccount(requst *NewAccountRequest, metadata Metadata, ch chan ImportResponse) {
	ch <- ImportResponse{true, "", ""}
}
func (ui *HeadlessUI) ShowError(message string) {
	//stdout is used by communication
	fmt.Fprint(os.Stderr, message)
}
func (ui *HeadlessUI) ShowInfo(message string) {
	//stdout is used by communication
	fmt.Fprint(os.Stderr, message)
}

// NewSignerAPI creates a new API that can be used for account management.
// ksLocation specifies the directory where to store the password protected private
// key that is generated when a new account is created.
// noUSB disables USB support that is required to support hardware devices such as
// ledger and trezor.
func NewSignerAPI(chainID int64, ksLocation string, noUSB bool, ui SignerUI) *SignerAPI {
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

	return &SignerAPI{big.NewInt(chainID), accounts.NewManager(backends...), ui}
}

func metaData(ctx context.Context) Metadata {
	m := Metadata{"NA", "NA", "NA"}

	if v := ctx.Value("remote"); v != nil {
		m.remote = v.(string)
	}
	if v := ctx.Value("scheme"); v != nil {
		m.scheme = v.(string)
	}
	if v := ctx.Value("local"); v != nil {
		m.local = v.(string)
	}
	return m
}

// List returns the set of wallet this signer manages. Each wallet can contain
// multiple accounts.
func (api *SignerAPI) List(ctx context.Context) ([]Account, error) {

	ch := make(chan ListResponse, 1)

	var accounts []Account
	for _, wallet := range api.am.Wallets() {
		for _, acc := range wallet.Accounts() {
			acc := Account{Typ: "account", URL: wallet.URL(), Address: acc.Address}
			accounts = append(accounts, acc)
		}
	}

	api.ui.ApproveListing(&ListRequest{accounts: accounts}, metaData(ctx), ch)
	if result := <-ch; result.accounts != nil {
		return result.accounts, nil
	}
	return nil, fmt.Errorf("Listing denied")
}

// New creates a new password protected account. The private key is protected with
// the given password. Users are responsible to backup the private key that is stored
// in the keystore location thas was specified when this API was created.
func (api *SignerAPI) New(ctx context.Context, passphrase string) (accounts.Account, error) {
	be := api.am.Backends(keystore.KeyStoreType)
	if len(be) == 0 {
		return accounts.Account{}, errors.New("password based accounts not supported")
	}
	ch := make(chan NewAccountResponse, 1)
	api.ui.ApproveNewAccount(&NewAccountRequest{}, metaData(ctx), ch)

	if resp := <-ch; resp.approved {
		return be[0].(*keystore.KeyStore).NewAccount(passphrase)
	}
	return accounts.Account{}, fmt.Errorf("Request denied")
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

	req := SignTxRequest{transaction: tx, from: acc}
	if len(tx.Data()) > 3 {
		var abidef string

		// Try to make sense of the data
		abidef, err = lookupABI(tx.Data()[:4])
		if err != nil {
			req.callinfo = errorWrapper{"Warning! Could not locate ABI", err}
		} else {
			req.callinfo, err = parseCallData(tx.Data(), abidef)
			if err != nil {
				req.callinfo = errorWrapper{"Warning! Could not validate ABI-data against calldata", err}
			}
		}
	}

	ch := make(chan SignTxResponse, 1)
	api.ui.ApproveTx(&req, metaData(ctx), ch)

	if result := <-ch; result.approved {
		//Sanity check
		if result.hash != tx.Hash() {
			return nil, fmt.Errorf("Transaction hash mismatch")
		}
		signedTx, err := wallet.SignTxWithPassphrase(acc, passwd, tx, api.chainID)
		if err != nil {
			return nil, err
		}
		return rlp.EncodeToBytes(signedTx)
	}

	return nil, fmt.Errorf("Transaction rejected")

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
	ch := make(chan SignDataResponse, 1)

	sighash, msg := signHash(data)

	api.ui.ApproveSignData(&SignDataRequest{account: account, rawdata: data, message: msg, hash: sighash}, metaData(ctx), ch)

	if (<-ch).approved {

		// Assemble sign the data with the wallet
		signature, err := wallet.SignHashWithPassphrase(account, passwd, sighash)
		if err != nil {
			return nil, err
		}
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
		return signature, nil

	}
	return nil, fmt.Errorf("Signing rejected")
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

	hash, _ := signHash(data)
	rpk, err := crypto.Ecrecover(hash, sig)
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
func signHash(data []byte) ([]byte, string) {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg)), msg
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
	ch := make(chan ExportResponse, 1)

	api.ui.ApproveExport(&ExportRequest{account: account, file: url.Path}, metaData(ctx), ch)

	if (<-ch).approved {
		return ioutil.ReadFile(url.Path)
	}
	return nil, fmt.Errorf("Export rejected")
}

// Imports tries to import the given keyJSON in the local keystore. The keyJSON data is expected to be
// in web3 keystore format. It will decrypt the keyJSON with the given passphrase and on successful
// decryption it will encrypt the key with the given newPassphrase and store it in the keystore.
func (api *SignerAPI) Import(ctx context.Context, keyJSON json.RawMessage, passphrase, newPassphrase string) (Account, error) {
	be := api.am.Backends(keystore.KeyStoreType)

	if len(be) == 0 {
		return Account{}, errors.New("password based accounts not supported")
	}

	ch := make(chan ImportResponse, 1)

	api.ui.ApproveImport(&ImportRequest{}, metaData(ctx), ch)
	if resp := <-ch; resp.approved {

		acc, err := be[0].(*keystore.KeyStore).Import(keyJSON, passphrase, newPassphrase)
		if err != nil {
			return Account{}, err
		}

		return Account{Typ: "account", URL: acc.URL, Address: acc.Address}, nil
	}
	return Account{}, fmt.Errorf("Import rejected")

}
