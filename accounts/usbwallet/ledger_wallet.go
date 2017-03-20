// Copyright 2017 The go-ethereum Authors
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

// This file contains the implementation for interacting with the Ledger hardware
// wallets. The wire protocol spec can be found in the Ledger Blue GitHub repo:
// https://raw.githubusercontent.com/LedgerHQ/blue-app-eth/master/doc/ethapp.asc

package usbwallet

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/karalabe/hid"
)

// Maximum time between wallet health checks to detect USB unplugs.
const ledgerHeartbeatCycle = time.Second

// Minimum time to wait between self derivation attempts, even it the user is
// requesting accounts like crazy.
const ledgerSelfDeriveThrottling = time.Second

// ledgerOpcode is an enumeration encoding the supported Ledger opcodes.
type ledgerOpcode byte

// ledgerParam1 is an enumeration encoding the supported Ledger parameters for
// specific opcodes. The same parameter values may be reused between opcodes.
type ledgerParam1 byte

// ledgerParam2 is an enumeration encoding the supported Ledger parameters for
// specific opcodes. The same parameter values may be reused between opcodes.
type ledgerParam2 byte

const (
	ledgerOpRetrieveAddress  ledgerOpcode = 0x02 // Returns the public key and Ethereum address for a given BIP 32 path
	ledgerOpSignTransaction  ledgerOpcode = 0x04 // Signs an Ethereum transaction after having the user validate the parameters
	ledgerOpGetConfiguration ledgerOpcode = 0x06 // Returns specific wallet application configuration

	ledgerP1DirectlyFetchAddress    ledgerParam1 = 0x00 // Return address directly from the wallet
	ledgerP1ConfirmFetchAddress     ledgerParam1 = 0x01 // Require a user confirmation before returning the address
	ledgerP1InitTransactionData     ledgerParam1 = 0x00 // First transaction data block for signing
	ledgerP1ContTransactionData     ledgerParam1 = 0x80 // Subsequent transaction data block for signing
	ledgerP2DiscardAddressChainCode ledgerParam2 = 0x00 // Do not return the chain code along with the address
	ledgerP2ReturnAddressChainCode  ledgerParam2 = 0x01 // Require a user confirmation before returning the address
)

// errReplyInvalidHeader is the error message returned by a Ledger data exchange
// if the device replies with a mismatching header. This usually means the device
// is in browser mode.
var errReplyInvalidHeader = errors.New("invalid reply header")

// errInvalidVersionReply is the error message returned by a Ledger version retrieval
// when a response does arrive, but it does not contain the expected data.
var errInvalidVersionReply = errors.New("invalid version reply")

// ledgerWallet represents a live USB Ledger hardware wallet.
type ledgerWallet struct {
	hub *LedgerHub    // USB hub the device originates from (TODO(karalabe): remove if hotplug lands on Windows)
	url *accounts.URL // Textual URL uniquely identifying this wallet

	info    hid.DeviceInfo // Known USB device infos about the wallet
	device  *hid.Device    // USB device advertising itself as a Ledger wallet
	failure error          // Any failure that would make the device unusable

	version  [3]byte                                    // Current version of the Ledger Ethereum app (zero if app is offline)
	browser  bool                                       // Flag whether the Ledger is in browser mode (reply channel mismatch)
	accounts []accounts.Account                         // List of derive accounts pinned on the Ledger
	paths    map[common.Address]accounts.DerivationPath // Known derivation paths for signing operations

	deriveNextPath accounts.DerivationPath   // Next derivation path for account auto-discovery
	deriveNextAddr common.Address            // Next derived account address for auto-discovery
	deriveChain    ethereum.ChainStateReader // Blockchain state reader to discover used account with
	deriveReq      chan chan struct{}        // Channel to request a self-derivation on
	deriveQuit     chan chan error           // Channel to terminate the self-deriver with

	healthQuit chan chan error

	// Locking a hardware wallet is a bit special. Since hardware devices are lower
	// performing, any communication with them might take a non negligible amount of
	// time. Worse still, waiting for user confirmation can take arbitrarily long,
	// but exclusive communication must be upheld during. Locking the entire wallet
	// in the mean time however would stall any parts of the system that don't want
	// to communicate, just read some state (e.g. list the accounts).
	//
	// As such, a hardware wallet needs two locks to function correctly. A state
	// lock can be used to protect the wallet's software-side internal state, which
	// must not be held exlusively during hardware communication. A communication
	// lock can be used to achieve exclusive access to the device itself, this one
	// however should allow "skipping" waiting for operations that might want to
	// use the device, but can live without too (e.g. account self-derivation).
	//
	// Since we have two locks, it's important to know how to properly use them:
	//   - Communication requires the `device` to not change, so obtaining the
	//     commsLock should be done after having a stateLock.
	//   - Communication must not disable read access to the wallet state, so it
	//     must only ever hold a *read* lock to stateLock.
	commsLock chan struct{} // Mutex (buf=1) for the USB comms without keeping the state locked
	stateLock sync.RWMutex  // Protects read and write access to the wallet struct fields

	log log.Logger // Contextual logger to tag the ledger with its id
}

// URL implements accounts.Wallet, returning the URL of the Ledger device.
func (w *ledgerWallet) URL() accounts.URL {
	return *w.url // Immutable, no need for a lock
}

// Status implements accounts.Wallet, always whether the Ledger is opened, closed
// or whether the Ethereum app was not started on it.
func (w *ledgerWallet) Status() string {
	w.stateLock.RLock() // No device communication, state lock is enough
	defer w.stateLock.RUnlock()

	if w.failure != nil {
		return fmt.Sprintf("Failed: %v", w.failure)
	}
	if w.device == nil {
		return "Closed"
	}
	if w.browser {
		return "Ethereum app in browser mode"
	}
	if w.offline() {
		return "Ethereum app offline"
	}
	return fmt.Sprintf("Ethereum app v%d.%d.%d online", w.version[0], w.version[1], w.version[2])
}

// offline returns whether the wallet and the Ethereum app is offline or not.
//
// The method assumes that the state lock is held!
func (w *ledgerWallet) offline() bool {
	return w.version == [3]byte{0, 0, 0}
}

// failed returns if the USB device wrapped by the wallet failed for some reason.
// This is used by the device scanner to report failed wallets as departed.
//
// The method assumes that the state lock is *not* held!
func (w *ledgerWallet) failed() bool {
	w.stateLock.RLock() // No device communication, state lock is enough
	defer w.stateLock.RUnlock()

	return w.failure != nil
}

// Open implements accounts.Wallet, attempting to open a USB connection to the
// Ledger hardware wallet. The Ledger does not require a user passphrase, so that
// parameter is silently discarded.
func (w *ledgerWallet) Open(passphrase string) error {
	w.stateLock.Lock() // State lock is enough since there's no connection yet at this point
	defer w.stateLock.Unlock()

	// If the wallet was already opened, don't try to open again
	if w.device != nil {
		return accounts.ErrWalletAlreadyOpen
	}
	// Otherwise iterate over all USB devices and find this again (no way to directly do this)
	device, err := w.info.Open()
	if err != nil {
		return err
	}
	// Wallet seems to be successfully opened, guess if the Ethereum app is running
	w.device = device
	w.commsLock = make(chan struct{}, 1)
	w.commsLock <- struct{}{} // Enable lock

	w.paths = make(map[common.Address]accounts.DerivationPath)

	w.deriveReq = make(chan chan struct{})
	w.deriveQuit = make(chan chan error)
	w.healthQuit = make(chan chan error)

	defer func() {
		go w.heartbeat()
		go w.selfDerive()
	}()

	if _, err = w.ledgerDerive(accounts.DefaultBaseDerivationPath); err != nil {
		// Ethereum app is not running or in browser mode, nothing more to do, return
		if err == errReplyInvalidHeader {
			w.browser = true
		}
		return nil
	}
	// Try to resolve the Ethereum app's version, will fail prior to v1.0.2
	if w.version, err = w.ledgerVersion(); err != nil {
		w.version = [3]byte{1, 0, 0} // Assume worst case, can't verify if v1.0.0 or v1.0.1
	}
	return nil
}

// heartbeat is a health check loop for the Ledger wallets to periodically verify
// whether they are still present or if they malfunctioned. It is needed because:
//  - libusb on Windows doesn't support hotplug, so we can't detect USB unplugs
//  - communication timeout on the Ledger requires a device power cycle to fix
func (w *ledgerWallet) heartbeat() {
	w.log.Debug("Ledger health-check started")
	defer w.log.Debug("Ledger health-check stopped")

	// Execute heartbeat checks until termination or error
	var (
		errc chan error
		err  error
	)
	for errc == nil && err == nil {
		// Wait until termination is requested or the heartbeat cycle arrives
		select {
		case errc = <-w.healthQuit:
			// Termination requested
			continue
		case <-time.After(ledgerHeartbeatCycle):
			// Heartbeat time
		}
		// Execute a tiny data exchange to see responsiveness
		w.stateLock.RLock()
		if w.device == nil {
			// Terminated while waiting for the lock
			w.stateLock.RUnlock()
			continue
		}
		<-w.commsLock // Don't lock state while resolving version
		_, err = w.ledgerVersion()
		w.commsLock <- struct{}{}
		w.stateLock.RUnlock()

		if err != nil && err != errInvalidVersionReply {
			w.stateLock.Lock() // Lock state to tear the wallet down
			w.failure = err
			w.close()
			w.stateLock.Unlock()
		}
		// Ignore non hardware related errors
		err = nil
	}
	// In case of error, wait for termination
	if err != nil {
		w.log.Debug("Ledger health-check failed", "err", err)
		errc = <-w.healthQuit
	}
	errc <- err
}

// Close implements accounts.Wallet, closing the USB connection to the Ledger.
func (w *ledgerWallet) Close() error {
	// Ensure the wallet was opened
	w.stateLock.RLock()
	hQuit, dQuit := w.healthQuit, w.deriveQuit
	w.stateLock.RUnlock()

	// Terminate the health checks
	var herr error
	if hQuit != nil {
		errc := make(chan error)
		hQuit <- errc
		herr = <-errc // Save for later, we *must* close the USB
	}
	// Terminate the self-derivations
	var derr error
	if dQuit != nil {
		errc := make(chan error)
		dQuit <- errc
		derr = <-errc // Save for later, we *must* close the USB
	}
	// Terminate the device connection
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	w.healthQuit = nil
	w.deriveQuit = nil
	w.deriveReq = nil

	if err := w.close(); err != nil {
		return err
	}
	if herr != nil {
		return herr
	}
	return derr
}

// close is the internal wallet closer that terminates the USB connection and
// resets all the fields to their defaults.
//
// Note, close assumes the state lock is held!
func (w *ledgerWallet) close() error {
	// Allow duplicate closes, especially for health-check failures
	if w.device == nil {
		return nil
	}
	// Close the device, clear everything, then return
	w.device.Close()
	w.device = nil

	w.browser, w.version = false, [3]byte{}
	w.accounts, w.paths = nil, nil

	return nil
}

// Accounts implements accounts.Wallet, returning the list of accounts pinned to
// the Ledger hardware wallet. If self-derivation was enabled, the account list
// is periodically expanded based on current chain state.
func (w *ledgerWallet) Accounts() []accounts.Account {
	// Attempt self-derivation if it's running
	reqc := make(chan struct{}, 1)
	select {
	case w.deriveReq <- reqc:
		// Self-derivation request accepted, wait for it
		<-reqc
	default:
		// Self-derivation offline, throttled or busy, skip
	}
	// Return whatever account list we ended up with
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	cpy := make([]accounts.Account, len(w.accounts))
	copy(cpy, w.accounts)
	return cpy
}

// selfDerive is an account derivation loop that upon request attempts to find
// new non-zero accounts.
func (w *ledgerWallet) selfDerive() {
	w.log.Debug("Ledger self-derivation started")
	defer w.log.Debug("Ledger self-derivation stopped")

	// Execute self-derivations until termination or error
	var (
		reqc chan struct{}
		errc chan error
		err  error
	)
	for errc == nil && err == nil {
		// Wait until either derivation or termination is requested
		select {
		case errc = <-w.deriveQuit:
			// Termination requested
			continue
		case reqc = <-w.deriveReq:
			// Account discovery requested
		}
		// Derivation needs a chain and device access, skip if either unavailable
		w.stateLock.RLock()
		if w.device == nil || w.deriveChain == nil || w.offline() {
			w.stateLock.RUnlock()
			reqc <- struct{}{}
			continue
		}
		select {
		case <-w.commsLock:
		default:
			w.stateLock.RUnlock()
			reqc <- struct{}{}
			continue
		}
		// Device lock obtained, derive the next batch of accounts
		var (
			accs  []accounts.Account
			paths []accounts.DerivationPath

			nextAddr = w.deriveNextAddr
			nextPath = w.deriveNextPath

			context = context.Background()
		)
		for empty := false; !empty; {
			// Retrieve the next derived Ethereum account
			if nextAddr == (common.Address{}) {
				if nextAddr, err = w.ledgerDerive(nextPath); err != nil {
					w.log.Warn("Ledger account derivation failed", "err", err)
					break
				}
			}
			// Check the account's status against the current chain state
			var (
				balance *big.Int
				nonce   uint64
			)
			balance, err = w.deriveChain.BalanceAt(context, nextAddr, nil)
			if err != nil {
				w.log.Warn("Ledger balance retrieval failed", "err", err)
				break
			}
			nonce, err = w.deriveChain.NonceAt(context, nextAddr, nil)
			if err != nil {
				w.log.Warn("Ledger nonce retrieval failed", "err", err)
				break
			}
			// If the next account is empty, stop self-derivation, but add it nonetheless
			if balance.Sign() == 0 && nonce == 0 {
				empty = true
			}
			// We've just self-derived a new account, start tracking it locally
			path := make(accounts.DerivationPath, len(nextPath))
			copy(path[:], nextPath[:])
			paths = append(paths, path)

			account := accounts.Account{
				Address: nextAddr,
				URL:     accounts.URL{Scheme: w.url.Scheme, Path: fmt.Sprintf("%s/%s", w.url.Path, path)},
			}
			accs = append(accs, account)

			// Display a log message to the user for new (or previously empty accounts)
			if _, known := w.paths[nextAddr]; !known || (!empty && nextAddr == w.deriveNextAddr) {
				w.log.Info("Ledger discovered new account", "address", nextAddr, "path", path, "balance", balance, "nonce", nonce)
			}
			// Fetch the next potential account
			if !empty {
				nextAddr = common.Address{}
				nextPath[len(nextPath)-1]++
			}
		}
		// Self derivation complete, release device lock
		w.commsLock <- struct{}{}
		w.stateLock.RUnlock()

		// Insert any accounts successfully derived
		w.stateLock.Lock()
		for i := 0; i < len(accs); i++ {
			if _, ok := w.paths[accs[i].Address]; !ok {
				w.accounts = append(w.accounts, accs[i])
				w.paths[accs[i].Address] = paths[i]
			}
		}
		// Shift the self-derivation forward
		// TODO(karalabe): don't overwrite changes from wallet.SelfDerive
		w.deriveNextAddr = nextAddr
		w.deriveNextPath = nextPath
		w.stateLock.Unlock()

		// Notify the user of termination and loop after a bit of time (to avoid trashing)
		reqc <- struct{}{}
		if err == nil {
			select {
			case errc = <-w.deriveQuit:
				// Termination requested, abort
			case <-time.After(ledgerSelfDeriveThrottling):
				// Waited enough, willing to self-derive again
			}
		}
	}
	// In case of error, wait for termination
	if err != nil {
		w.log.Debug("Ledger self-derivation failed", "err", err)
		errc = <-w.deriveQuit
	}
	errc <- err
}

// Contains implements accounts.Wallet, returning whether a particular account is
// or is not pinned into this Ledger instance. Although we could attempt to resolve
// unpinned accounts, that would be an non-negligible hardware operation.
func (w *ledgerWallet) Contains(account accounts.Account) bool {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	_, exists := w.paths[account.Address]
	return exists
}

// Derive implements accounts.Wallet, deriving a new account at the specific
// derivation path. If pin is set to true, the account will be added to the list
// of tracked accounts.
func (w *ledgerWallet) Derive(path accounts.DerivationPath, pin bool) (accounts.Account, error) {
	// Try to derive the actual account and update its URL if successful
	w.stateLock.RLock() // Avoid device disappearing during derivation

	if w.device == nil || w.offline() {
		w.stateLock.RUnlock()
		return accounts.Account{}, accounts.ErrWalletClosed
	}
	<-w.commsLock // Avoid concurrent hardware access
	address, err := w.ledgerDerive(path)
	w.commsLock <- struct{}{}

	w.stateLock.RUnlock()

	// If an error occurred or no pinning was requested, return
	if err != nil {
		return accounts.Account{}, err
	}
	account := accounts.Account{
		Address: address,
		URL:     accounts.URL{Scheme: w.url.Scheme, Path: fmt.Sprintf("%s/%s", w.url.Path, path)},
	}
	if !pin {
		return account, nil
	}
	// Pinning needs to modify the state
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	if _, ok := w.paths[address]; !ok {
		w.accounts = append(w.accounts, account)
		w.paths[address] = path
	}
	return account, nil
}

// SelfDerive implements accounts.Wallet, trying to discover accounts that the
// user used previously (based on the chain state), but ones that he/she did not
// explicitly pin to the wallet manually. To avoid chain head monitoring, self
// derivation only runs during account listing (and even then throttled).
func (w *ledgerWallet) SelfDerive(base accounts.DerivationPath, chain ethereum.ChainStateReader) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	w.deriveNextPath = make(accounts.DerivationPath, len(base))
	copy(w.deriveNextPath[:], base[:])

	w.deriveNextAddr = common.Address{}
	w.deriveChain = chain
}

// SignHash implements accounts.Wallet, however signing arbitrary data is not
// supported for Ledger wallets, so this method will always return an error.
func (w *ledgerWallet) SignHash(acc accounts.Account, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTx implements accounts.Wallet. It sends the transaction over to the Ledger
// wallet to request a confirmation from the user. It returns either the signed
// transaction or a failure if the user denied the transaction.
//
// Note, if the version of the Ethereum application running on the Ledger wallet is
// too old to sign EIP-155 transactions, but such is requested nonetheless, an error
// will be returned opposed to silently signing in Homestead mode.
func (w *ledgerWallet) SignTx(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	w.stateLock.RLock() // Comms have own mutex, this is for the state fields
	defer w.stateLock.RUnlock()

	// If the wallet is closed, or the Ethereum app doesn't run, abort
	if w.device == nil || w.offline() {
		return nil, accounts.ErrWalletClosed
	}
	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}
	// Ensure the wallet is capable of signing the given transaction
	if chainID != nil && w.version[0] <= 1 && w.version[1] <= 0 && w.version[2] <= 2 {
		return nil, fmt.Errorf("Ledger v%d.%d.%d doesn't support signing this transaction, please update to v1.0.3 at least", w.version[0], w.version[1], w.version[2])
	}
	// All infos gathered and metadata checks out, request signing
	<-w.commsLock
	defer func() { w.commsLock <- struct{}{} }()

	// Ensure the device isn't screwed with while user confirmation is pending
	// TODO(karalabe): remove if hotplug lands on Windows
	w.hub.commsLock.RLock()
	defer w.hub.commsLock.RUnlock()

	return w.ledgerSign(path, account.Address, tx, chainID)
}

// SignHashWithPassphrase implements accounts.Wallet, however signing arbitrary
// data is not supported for Ledger wallets, so this method will always return
// an error.
func (w *ledgerWallet) SignHashWithPassphrase(account accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTxWithPassphrase implements accounts.Wallet, attempting to sign the given
// transaction with the given account using passphrase as extra authentication.
// Since the Ledger does not support extra passphrases, it is silently ignored.
func (w *ledgerWallet) SignTxWithPassphrase(account accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return w.SignTx(account, tx, chainID)
}

// ledgerVersion retrieves the current version of the Ethereum wallet app running
// on the Ledger wallet.
//
// The version retrieval protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc | Le
//   ----+-----+----+----+----+---
//    E0 | 06  | 00 | 00 | 00 | 04
//
// With no input data, and the output data being:
//
//   Description                                        | Length
//   ---------------------------------------------------+--------
//   Flags 01: arbitrary data signature enabled by user | 1 byte
//   Application major version                          | 1 byte
//   Application minor version                          | 1 byte
//   Application patch version                          | 1 byte
func (w *ledgerWallet) ledgerVersion() ([3]byte, error) {
	// Send the request and wait for the response
	reply, err := w.ledgerExchange(ledgerOpGetConfiguration, 0, 0, nil)
	if err != nil {
		return [3]byte{}, err
	}
	if len(reply) != 4 {
		return [3]byte{}, errInvalidVersionReply
	}
	// Cache the version for future reference
	var version [3]byte
	copy(version[:], reply[1:])
	return version, nil
}

// ledgerDerive retrieves the currently active Ethereum address from a Ledger
// wallet at the specified derivation path.
//
// The address derivation protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc  | Le
//   ----+-----+----+----+-----+---
//    E0 | 02  | 00 return address
//               01 display address and confirm before returning
//                  | 00: do not return the chain code
//                  | 01: return the chain code
//                       | var | 00
//
// Where the input data is:
//
//   Description                                      | Length
//   -------------------------------------------------+--------
//   Number of BIP 32 derivations to perform (max 10) | 1 byte
//   First derivation index (big endian)              | 4 bytes
//   ...                                              | 4 bytes
//   Last derivation index (big endian)               | 4 bytes
//
// And the output data is:
//
//   Description             | Length
//   ------------------------+-------------------
//   Public Key length       | 1 byte
//   Uncompressed Public Key | arbitrary
//   Ethereum address length | 1 byte
//   Ethereum address        | 40 bytes hex ascii
//   Chain code if requested | 32 bytes
func (w *ledgerWallet) ledgerDerive(derivationPath []uint32) (common.Address, error) {
	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(derivationPath))
	path[0] = byte(len(derivationPath))
	for i, component := range derivationPath {
		binary.BigEndian.PutUint32(path[1+4*i:], component)
	}
	// Send the request and wait for the response
	reply, err := w.ledgerExchange(ledgerOpRetrieveAddress, ledgerP1DirectlyFetchAddress, ledgerP2DiscardAddressChainCode, path)
	if err != nil {
		return common.Address{}, err
	}
	// Discard the public key, we don't need that for now
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return common.Address{}, errors.New("reply lacks public key entry")
	}
	reply = reply[1+int(reply[0]):]

	// Extract the Ethereum hex address string
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return common.Address{}, errors.New("reply lacks address entry")
	}
	hexstr := reply[1 : 1+int(reply[0])]

	// Decode the hex sting into an Ethereum address and return
	var address common.Address
	hex.Decode(address[:], hexstr)
	return address, nil
}

// ledgerSign sends the transaction to the Ledger wallet, and waits for the user
// to confirm or deny the transaction.
//
// The transaction signing protocol is defined as follows:
//
//   CLA | INS | P1 | P2 | Lc  | Le
//   ----+-----+----+----+-----+---
//    E0 | 04  | 00: first transaction data block
//               80: subsequent transaction data block
//                  | 00 | variable | variable
//
// Where the input for the first transaction block (first 255 bytes) is:
//
//   Description                                      | Length
//   -------------------------------------------------+----------
//   Number of BIP 32 derivations to perform (max 10) | 1 byte
//   First derivation index (big endian)              | 4 bytes
//   ...                                              | 4 bytes
//   Last derivation index (big endian)               | 4 bytes
//   RLP transaction chunk                            | arbitrary
//
// And the input for subsequent transaction blocks (first 255 bytes) are:
//
//   Description           | Length
//   ----------------------+----------
//   RLP transaction chunk | arbitrary
//
// And the output data is:
//
//   Description | Length
//   ------------+---------
//   signature V | 1 byte
//   signature R | 32 bytes
//   signature S | 32 bytes
func (w *ledgerWallet) ledgerSign(derivationPath []uint32, address common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(derivationPath))
	path[0] = byte(len(derivationPath))
	for i, component := range derivationPath {
		binary.BigEndian.PutUint32(path[1+4*i:], component)
	}
	// Create the transaction RLP based on whether legacy or EIP155 signing was requeste
	var (
		txrlp []byte
		err   error
	)
	if chainID == nil {
		if txrlp, err = rlp.EncodeToBytes([]interface{}{tx.Nonce(), tx.GasPrice(), tx.Gas(), tx.To(), tx.Value(), tx.Data()}); err != nil {
			return nil, err
		}
	} else {
		if txrlp, err = rlp.EncodeToBytes([]interface{}{tx.Nonce(), tx.GasPrice(), tx.Gas(), tx.To(), tx.Value(), tx.Data(), chainID, big.NewInt(0), big.NewInt(0)}); err != nil {
			return nil, err
		}
	}
	payload := append(path, txrlp...)

	// Send the request and wait for the response
	var (
		op    = ledgerP1InitTransactionData
		reply []byte
	)
	for len(payload) > 0 {
		// Calculate the size of the next data chunk
		chunk := 255
		if chunk > len(payload) {
			chunk = len(payload)
		}
		// Send the chunk over, ensuring it's processed correctly
		reply, err = w.ledgerExchange(ledgerOpSignTransaction, op, 0, payload[:chunk])
		if err != nil {
			return nil, err
		}
		// Shift the payload and ensure subsequent chunks are marked as such
		payload = payload[chunk:]
		op = ledgerP1ContTransactionData
	}
	// Extract the Ethereum signature and do a sanity validation
	if len(reply) != 65 {
		return nil, errors.New("reply lacks signature")
	}
	signature := append(reply[1:], reply[0])

	// Create the correct signer and signature transform based on the chain ID
	var signer types.Signer
	if chainID == nil {
		signer = new(types.HomesteadSigner)
	} else {
		signer = types.NewEIP155Signer(chainID)
		signature[64] = signature[64] - byte(chainID.Uint64()*2+35)
	}
	// Inject the final signature into the transaction and sanity check the sender
	signed, err := tx.WithSignature(signer, signature)
	if err != nil {
		return nil, err
	}
	sender, err := types.Sender(signer, signed)
	if err != nil {
		return nil, err
	}
	if sender != address {
		return nil, fmt.Errorf("signer mismatch: expected %s, got %s", address.Hex(), sender.Hex())
	}
	return signed, nil
}

// ledgerExchange performs a data exchange with the Ledger wallet, sending it a
// message and retrieving the response.
//
// The common transport header is defined as follows:
//
//  Description                           | Length
//  --------------------------------------+----------
//  Communication channel ID (big endian) | 2 bytes
//  Command tag                           | 1 byte
//  Packet sequence index (big endian)    | 2 bytes
//  Payload                               | arbitrary
//
// The Communication channel ID allows commands multiplexing over the same
// physical link. It is not used for the time being, and should be set to 0101
// to avoid compatibility issues with implementations ignoring a leading 00 byte.
//
// The Command tag describes the message content. Use TAG_APDU (0x05) for standard
// APDU payloads, or TAG_PING (0x02) for a simple link test.
//
// The Packet sequence index describes the current sequence for fragmented payloads.
// The first fragment index is 0x00.
//
// APDU Command payloads are encoded as follows:
//
//  Description              | Length
//  -----------------------------------
//  APDU length (big endian) | 2 bytes
//  APDU CLA                 | 1 byte
//  APDU INS                 | 1 byte
//  APDU P1                  | 1 byte
//  APDU P2                  | 1 byte
//  APDU length              | 1 byte
//  Optional APDU data       | arbitrary
func (w *ledgerWallet) ledgerExchange(opcode ledgerOpcode, p1 ledgerParam1, p2 ledgerParam2, data []byte) ([]byte, error) {
	// Construct the message payload, possibly split into multiple chunks
	apdu := make([]byte, 2, 7+len(data))

	binary.BigEndian.PutUint16(apdu, uint16(5+len(data)))
	apdu = append(apdu, []byte{0xe0, byte(opcode), byte(p1), byte(p2), byte(len(data))}...)
	apdu = append(apdu, data...)

	// Stream all the chunks to the device
	header := []byte{0x01, 0x01, 0x05, 0x00, 0x00} // Channel ID and command tag appended
	chunk := make([]byte, 64)
	space := len(chunk) - len(header)

	for i := 0; len(apdu) > 0; i++ {
		// Construct the new message to stream
		chunk = append(chunk[:0], header...)
		binary.BigEndian.PutUint16(chunk[3:], uint16(i))

		if len(apdu) > space {
			chunk = append(chunk, apdu[:space]...)
			apdu = apdu[space:]
		} else {
			chunk = append(chunk, apdu...)
			apdu = nil
		}
		// Send over to the device
		w.log.Trace("Data chunk sent to the Ledger", "chunk", hexutil.Bytes(chunk))
		if _, err := w.device.Write(chunk); err != nil {
			return nil, err
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var reply []byte
	chunk = chunk[:64] // Yeah, we surely have enough space
	for {
		// Read the next chunk from the Ledger wallet
		if _, err := io.ReadFull(w.device, chunk); err != nil {
			return nil, err
		}
		w.log.Trace("Data chunk received from the Ledger", "chunk", hexutil.Bytes(chunk))

		// Make sure the transport header matches
		if chunk[0] != 0x01 || chunk[1] != 0x01 || chunk[2] != 0x05 {
			return nil, errReplyInvalidHeader
		}
		// If it's the first chunk, retrieve the total message length
		var payload []byte

		if chunk[3] == 0x00 && chunk[4] == 0x00 {
			reply = make([]byte, 0, int(binary.BigEndian.Uint16(chunk[5:7])))
			payload = chunk[7:]
		} else {
			payload = chunk[5:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}
	}
	return reply[:len(reply)-2], nil
}
