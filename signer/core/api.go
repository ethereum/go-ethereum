// Copyright 2018 The go-ethereum Authors
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

package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/signer/storage"
)

const (
	// numberOfAccountsToDerive For hardware wallets, the number of accounts to derive
	numberOfAccountsToDerive = 10
	// ExternalAPIVersion -- see extapi_changelog.md
	ExternalAPIVersion = "6.0.0"
	// InternalAPIVersion -- see intapi_changelog.md
	InternalAPIVersion = "7.0.0"
)

// ExternalAPI defines the external API through which signing requests are made.
type ExternalAPI interface {
	// List available accounts
	List(ctx context.Context) ([]common.Address, error)
	// New request to create a new account
	New(ctx context.Context) (common.Address, error)
	// SignTransaction request to sign the specified transaction
	SignTransaction(ctx context.Context, args SendTxArgs, methodSelector *string) (*ethapi.SignTransactionResult, error)
	// SignData - request to sign the given data (plus prefix)
	SignData(ctx context.Context, contentType string, addr common.MixedcaseAddress, data interface{}) (hexutil.Bytes, error)
	// SignTypedData - request to sign the given structured data (plus prefix)
	SignTypedData(ctx context.Context, addr common.MixedcaseAddress, data TypedData) (hexutil.Bytes, error)
	// EcRecover - recover public key from given message and signature
	EcRecover(ctx context.Context, data hexutil.Bytes, sig hexutil.Bytes) (common.Address, error)
	// Version info about the APIs
	Version(ctx context.Context) (string, error)
}

// UIClientAPI specifies what method a UI needs to implement to be able to be used as a
// UI for the signer
type UIClientAPI interface {
	// ApproveTx prompt the user for confirmation to request to sign Transaction
	ApproveTx(request *SignTxRequest) (SignTxResponse, error)
	// ApproveSignData prompt the user for confirmation to request to sign data
	ApproveSignData(request *SignDataRequest) (SignDataResponse, error)
	// ApproveListing prompt the user for confirmation to list accounts
	// the list of accounts to list can be modified by the UI
	ApproveListing(request *ListRequest) (ListResponse, error)
	// ApproveNewAccount prompt the user for confirmation to create new Account, and reveal to caller
	ApproveNewAccount(request *NewAccountRequest) (NewAccountResponse, error)
	// ShowError displays error message to user
	ShowError(message string)
	// ShowInfo displays info message to user
	ShowInfo(message string)
	// OnApprovedTx notifies the UI about a transaction having been successfully signed.
	// This method can be used by a UI to keep track of e.g. how much has been sent to a particular recipient.
	OnApprovedTx(tx ethapi.SignTransactionResult)
	// OnSignerStartup is invoked when the signer boots, and tells the UI info about external API location and version
	// information
	OnSignerStartup(info StartupInfo)
	// OnInputRequired is invoked when clef requires user input, for example master password or
	// pin-code for unlocking hardware wallets
	OnInputRequired(info UserInputRequest) (UserInputResponse, error)
	// RegisterUIServer tells the UI to use the given UIServerAPI for ui->clef communication
	RegisterUIServer(api *UIServerAPI)
}

// Validator defines the methods required to validate a transaction against some
// sanity defaults as well as any underlying 4byte method database.
//
// Use fourbyte.Database as an implementation. It is separated out of this package
// to allow pieces of the signer package to be used without having to load the
// 7MB embedded 4byte dump.
type Validator interface {
	// ValidateTransaction does a number of checks on the supplied transaction, and
	// returns either a list of warnings, or an error (indicating that the transaction
	// should be immediately rejected).
	ValidateTransaction(selector *string, tx *SendTxArgs) (*ValidationMessages, error)
}

// SignerAPI defines the actual implementation of ExternalAPI
type SignerAPI struct {
	chainID     *big.Int
	am          *accounts.Manager
	UI          UIClientAPI
	validator   Validator
	rejectMode  bool
	credentials storage.Storage
}

// Metadata about a request
type Metadata struct {
	Remote    string `json:"remote"`
	Local     string `json:"local"`
	Scheme    string `json:"scheme"`
	UserAgent string `json:"User-Agent"`
	Origin    string `json:"Origin"`
}

func StartClefAccountManager(ksLocation string, nousb, lightKDF bool, scpath string) *accounts.Manager {
	var (
		backends []accounts.Backend
		n, p     = keystore.StandardScryptN, keystore.StandardScryptP
	)
	if lightKDF {
		n, p = keystore.LightScryptN, keystore.LightScryptP
	}
	// support password based accounts
	if len(ksLocation) > 0 {
		backends = append(backends, keystore.NewKeyStore(ksLocation, n, p))
	}
	if !nousb {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			backends = append(backends, ledgerhub)
			log.Debug("Ledger support enabled")
		}
		// Start a USB hub for Trezor hardware wallets (HID version)
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			backends = append(backends, trezorhub)
			log.Debug("Trezor support enabled via HID")
		}
		// Start a USB hub for Trezor hardware wallets (WebUSB version)
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			backends = append(backends, trezorhub)
			log.Debug("Trezor support enabled via WebUSB")
		}
	}

	// Start a smart card hub
	if len(scpath) > 0 {
		// Sanity check that the smartcard path is valid
		fi, err := os.Stat(scpath)
		if err != nil {
			log.Info("Smartcard socket file missing, disabling", "err", err)
		} else {
			if fi.Mode()&os.ModeType != os.ModeSocket {
				log.Error("Invalid smartcard socket file type", "path", scpath, "type", fi.Mode().String())
			} else {
				if schub, err := scwallet.NewHub(scpath, scwallet.Scheme, ksLocation); err != nil {
					log.Warn(fmt.Sprintf("Failed to start smart card hub, disabling: %v", err))
				} else {
					backends = append(backends, schub)
				}
			}
		}
	}

	// Clef doesn't allow insecure http account unlock.
	return accounts.NewManager(&accounts.Config{InsecureUnlockAllowed: false}, backends...)
}

// MetadataFromContext extracts Metadata from a given context.Context
func MetadataFromContext(ctx context.Context) Metadata {
	m := Metadata{"NA", "NA", "NA", "", ""} // batman

	if v := ctx.Value("remote"); v != nil {
		m.Remote = v.(string)
	}
	if v := ctx.Value("scheme"); v != nil {
		m.Scheme = v.(string)
	}
	if v := ctx.Value("local"); v != nil {
		m.Local = v.(string)
	}
	if v := ctx.Value("Origin"); v != nil {
		m.Origin = v.(string)
	}
	if v := ctx.Value("User-Agent"); v != nil {
		m.UserAgent = v.(string)
	}
	return m
}

// String implements Stringer interface
func (m Metadata) String() string {
	s, err := json.Marshal(m)
	if err == nil {
		return string(s)
	}
	return err.Error()
}

// types for the requests/response types between signer and UI
type (
	// SignTxRequest contains info about a Transaction to sign
	SignTxRequest struct {
		Transaction SendTxArgs       `json:"transaction"`
		Callinfo    []ValidationInfo `json:"call_info"`
		Meta        Metadata         `json:"meta"`
	}
	// SignTxResponse result from SignTxRequest
	SignTxResponse struct {
		//The UI may make changes to the TX
		Transaction SendTxArgs `json:"transaction"`
		Approved    bool       `json:"approved"`
	}
	SignDataRequest struct {
		ContentType string                  `json:"content_type"`
		Address     common.MixedcaseAddress `json:"address"`
		Rawdata     []byte                  `json:"raw_data"`
		Messages    []*NameValueType        `json:"messages"`
		Hash        hexutil.Bytes           `json:"hash"`
		Meta        Metadata                `json:"meta"`
	}
	SignDataResponse struct {
		Approved bool `json:"approved"`
	}
	NewAccountRequest struct {
		Meta Metadata `json:"meta"`
	}
	NewAccountResponse struct {
		Approved bool `json:"approved"`
	}
	ListRequest struct {
		Accounts []accounts.Account `json:"accounts"`
		Meta     Metadata           `json:"meta"`
	}
	ListResponse struct {
		Accounts []accounts.Account `json:"accounts"`
	}
	Message struct {
		Text string `json:"text"`
	}
	StartupInfo struct {
		Info map[string]interface{} `json:"info"`
	}
	UserInputRequest struct {
		Title      string `json:"title"`
		Prompt     string `json:"prompt"`
		IsPassword bool   `json:"isPassword"`
	}
	UserInputResponse struct {
		Text string `json:"text"`
	}
)

var ErrRequestDenied = errors.New("Request denied")

// NewSignerAPI creates a new API that can be used for Account management.
// ksLocation specifies the directory where to store the password protected private
// key that is generated when a new Account is created.
// noUSB disables USB support that is required to support hardware devices such as
// ledger and trezor.
func NewSignerAPI(am *accounts.Manager, chainID int64, noUSB bool, ui UIClientAPI, validator Validator, advancedMode bool, credentials storage.Storage) *SignerAPI {
	if advancedMode {
		log.Info("Clef is in advanced mode: will warn instead of reject")
	}
	signer := &SignerAPI{big.NewInt(chainID), am, ui, validator, !advancedMode, credentials}
	if !noUSB {
		signer.startUSBListener()
	}
	return signer
}
func (api *SignerAPI) openTrezor(url accounts.URL) {
	resp, err := api.UI.OnInputRequired(UserInputRequest{
		Prompt: "Pin required to open Trezor wallet\n" +
			"Look at the device for number positions\n\n" +
			"7 | 8 | 9\n" +
			"--+---+--\n" +
			"4 | 5 | 6\n" +
			"--+---+--\n" +
			"1 | 2 | 3\n\n",
		IsPassword: true,
		Title:      "Trezor unlock",
	})
	if err != nil {
		log.Warn("failed getting trezor pin", "err", err)
		return
	}
	// We're using the URL instead of the pointer to the
	// Wallet -- perhaps it is not actually present anymore
	w, err := api.am.Wallet(url.String())
	if err != nil {
		log.Warn("wallet unavailable", "url", url)
		return
	}
	err = w.Open(resp.Text)
	if err != nil {
		log.Warn("failed to open wallet", "wallet", url, "err", err)
		return
	}

}

// startUSBListener starts a listener for USB events, for hardware wallet interaction
func (api *SignerAPI) startUSBListener() {
	events := make(chan accounts.WalletEvent, 16)
	am := api.am
	am.Subscribe(events)
	go func() {

		// Open any wallets already attached
		for _, wallet := range am.Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
				if err == usbwallet.ErrTrezorPINNeeded {
					go api.openTrezor(wallet.URL())
				}
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
					if err == usbwallet.ErrTrezorPINNeeded {
						go api.openTrezor(event.Wallet.URL())
					}
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				// Derive first N accounts, hardcoded for now
				var nextPath = make(accounts.DerivationPath, len(accounts.DefaultBaseDerivationPath))
				copy(nextPath[:], accounts.DefaultBaseDerivationPath[:])

				for i := 0; i < numberOfAccountsToDerive; i++ {
					acc, err := event.Wallet.Derive(nextPath, true)
					if err != nil {
						log.Warn("account derivation failed", "error", err)
					} else {
						log.Info("derived account", "address", acc.Address)
					}
					nextPath[len(nextPath)-1]++
				}
			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()
}

// List returns the set of wallet this signer manages. Each wallet can contain
// multiple accounts.
func (api *SignerAPI) List(ctx context.Context) ([]common.Address, error) {
	var accs []accounts.Account
	for _, wallet := range api.am.Wallets() {
		accs = append(accs, wallet.Accounts()...)
	}
	result, err := api.UI.ApproveListing(&ListRequest{Accounts: accs, Meta: MetadataFromContext(ctx)})
	if err != nil {
		return nil, err
	}
	if result.Accounts == nil {
		return nil, ErrRequestDenied

	}
	addresses := make([]common.Address, 0)
	for _, acc := range result.Accounts {
		addresses = append(addresses, acc.Address)
	}

	return addresses, nil
}

// New creates a new password protected Account. The private key is protected with
// the given password. Users are responsible to backup the private key that is stored
// in the keystore location thas was specified when this API was created.
func (api *SignerAPI) New(ctx context.Context) (common.Address, error) {
	be := api.am.Backends(keystore.KeyStoreType)
	if len(be) == 0 {
		return common.Address{}, errors.New("password based accounts not supported")
	}
	if resp, err := api.UI.ApproveNewAccount(&NewAccountRequest{MetadataFromContext(ctx)}); err != nil {
		return common.Address{}, err
	} else if !resp.Approved {
		return common.Address{}, ErrRequestDenied
	}

	// Three retries to get a valid password
	for i := 0; i < 3; i++ {
		resp, err := api.UI.OnInputRequired(UserInputRequest{
			"New account password",
			fmt.Sprintf("Please enter a password for the new account to be created (attempt %d of 3)", i),
			true})
		if err != nil {
			log.Warn("error obtaining password", "attempt", i, "error", err)
			continue
		}
		if pwErr := ValidatePasswordFormat(resp.Text); pwErr != nil {
			api.UI.ShowError(fmt.Sprintf("Account creation attempt #%d failed due to password requirements: %v", (i + 1), pwErr))
		} else {
			// No error
			acc, err := be[0].(*keystore.KeyStore).NewAccount(resp.Text)
			log.Info("Your new key was generated", "address", acc.Address)
			log.Warn("Please backup your key file!", "path", acc.URL.Path)
			log.Warn("Please remember your password!")
			return acc.Address, err
		}
	}
	// Otherwise fail, with generic error message
	return common.Address{}, errors.New("account creation failed")
}

// logDiff logs the difference between the incoming (original) transaction and the one returned from the signer.
// it also returns 'true' if the transaction was modified, to make it possible to configure the signer not to allow
// UI-modifications to requests
func logDiff(original *SignTxRequest, new *SignTxResponse) bool {
	modified := false
	if f0, f1 := original.Transaction.From, new.Transaction.From; !reflect.DeepEqual(f0, f1) {
		log.Info("Sender-account changed by UI", "was", f0, "is", f1)
		modified = true
	}
	if t0, t1 := original.Transaction.To, new.Transaction.To; !reflect.DeepEqual(t0, t1) {
		log.Info("Recipient-account changed by UI", "was", t0, "is", t1)
		modified = true
	}
	if g0, g1 := original.Transaction.Gas, new.Transaction.Gas; g0 != g1 {
		modified = true
		log.Info("Gas changed by UI", "was", g0, "is", g1)
	}
	if g0, g1 := big.Int(original.Transaction.GasPrice), big.Int(new.Transaction.GasPrice); g0.Cmp(&g1) != 0 {
		modified = true
		log.Info("GasPrice changed by UI", "was", g0, "is", g1)
	}
	if v0, v1 := big.Int(original.Transaction.Value), big.Int(new.Transaction.Value); v0.Cmp(&v1) != 0 {
		modified = true
		log.Info("Value changed by UI", "was", v0, "is", v1)
	}
	if d0, d1 := original.Transaction.Data, new.Transaction.Data; d0 != d1 {
		d0s := ""
		d1s := ""
		if d0 != nil {
			d0s = hexutil.Encode(*d0)
		}
		if d1 != nil {
			d1s = hexutil.Encode(*d1)
		}
		if d1s != d0s {
			modified = true
			log.Info("Data changed by UI", "was", d0s, "is", d1s)
		}
	}
	if n0, n1 := original.Transaction.Nonce, new.Transaction.Nonce; n0 != n1 {
		modified = true
		log.Info("Nonce changed by UI", "was", n0, "is", n1)
	}
	return modified
}

func (api *SignerAPI) lookupPassword(address common.Address) (string, error) {
	return api.credentials.Get(address.Hex())
}

func (api *SignerAPI) lookupOrQueryPassword(address common.Address, title, prompt string) (string, error) {
	// Look up the password and return if available
	if pw, err := api.lookupPassword(address); err == nil {
		return pw, nil
	}
	// Password unavailable, request it from the user
	pwResp, err := api.UI.OnInputRequired(UserInputRequest{title, prompt, true})
	if err != nil {
		log.Warn("error obtaining password", "error", err)
		// We'll not forward the error here, in case the error contains info about the response from the UI,
		// which could leak the password if it was malformed json or something
		return "", errors.New("internal error")
	}
	return pwResp.Text, nil
}

// SignTransaction signs the given Transaction and returns it both as json and rlp-encoded form
func (api *SignerAPI) SignTransaction(ctx context.Context, args SendTxArgs, methodSelector *string) (*ethapi.SignTransactionResult, error) {
	var (
		err    error
		result SignTxResponse
	)
	msgs, err := api.validator.ValidateTransaction(methodSelector, &args)
	if err != nil {
		return nil, err
	}
	// If we are in 'rejectMode', then reject rather than show the user warnings
	if api.rejectMode {
		if err := msgs.getWarnings(); err != nil {
			return nil, err
		}
	}
	req := SignTxRequest{
		Transaction: args,
		Meta:        MetadataFromContext(ctx),
		Callinfo:    msgs.Messages,
	}
	// Process approval
	result, err = api.UI.ApproveTx(&req)
	if err != nil {
		return nil, err
	}
	if !result.Approved {
		return nil, ErrRequestDenied
	}
	// Log changes made by the UI to the signing-request
	logDiff(&req, &result)
	var (
		acc    accounts.Account
		wallet accounts.Wallet
	)
	acc = accounts.Account{Address: result.Transaction.From.Address()}
	wallet, err = api.am.Find(acc)
	if err != nil {
		return nil, err
	}
	// Convert fields into a real transaction
	var unsignedTx = result.Transaction.toTransaction()
	// Get the password for the transaction
	pw, err := api.lookupOrQueryPassword(acc.Address, "Account password",
		fmt.Sprintf("Please enter the password for account %s", acc.Address.String()))
	if err != nil {
		return nil, err
	}
	// The one to sign is the one that was returned from the UI
	signedTx, err := wallet.SignTxWithPassphrase(acc, pw, unsignedTx, api.chainID)
	if err != nil {
		api.UI.ShowError(err.Error())
		return nil, err
	}

	rlpdata, err := rlp.EncodeToBytes(signedTx)
	response := ethapi.SignTransactionResult{Raw: rlpdata, Tx: signedTx}

	// Finally, send the signed tx to the UI
	api.UI.OnApprovedTx(response)
	// ...and to the external caller
	return &response, nil

}

// Returns the external api version. This method does not require user acceptance. Available methods are
// available via enumeration anyway, and this info does not contain user-specific data
func (api *SignerAPI) Version(ctx context.Context) (string, error) {
	return ExternalAPIVersion, nil
}
