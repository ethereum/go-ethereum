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

// This file contains the implementation for interacting with the Trezor hardware
// wallets. The wire protocol spec can be found on the SatoshiLabs website:
// https://doc.satoshilabs.com/trezor-tech/api-protobuf.html

package usbwallet

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/usbwallet/internal/trezor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/golang/protobuf/proto"
	"github.com/karalabe/hid"
)

// ErrTrezorPINNeeded is returned if opening the trezor requires a PIN code. In
// this case, the calling application should display a pinpad and send back the
// encoded passphrase.
var ErrTrezorPINNeeded = errors.New("trezor: pin needed")

// trezorWallet represents a live USB Trezor hardware wallet.
type trezorWallet struct {
	hub *TrezorHub    // USB hub the device originates from (TODO(karalabe): remove if hotplug lands on Windows)
	url *accounts.URL // Textual URL uniquely identifying this wallet

	info    hid.DeviceInfo // Known USB device infos about the wallet
	device  *hid.Device    // USB device advertising itself as a Trezor wallet
	failure error          // Any failure that would make the device unusable

	version  [3]uint32                                  // Current version of the Trezor formware (zero if app is offline)
	label    string                                     // Current textual label of the Trezor device
	pinwait  bool                                       // Flags whether the device is waiting for PIN entry
	accounts []accounts.Account                         // List of derive accounts pinned on the Trezor
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

	log log.Logger // Contextual logger to tag the trezor with its id
}

// URL implements accounts.Wallet, returning the URL of the Trezor device.
func (w *trezorWallet) URL() accounts.URL {
	return *w.url // Immutable, no need for a lock
}

// Status implements accounts.Wallet, always whether the Trezor is opened, closed
// or whether the Ethereum app was not started on it.
func (w *trezorWallet) Status() string {
	w.stateLock.RLock() // No device communication, state lock is enough
	defer w.stateLock.RUnlock()

	if w.failure != nil {
		return fmt.Sprintf("Failed: %v", w.failure)
	}
	if w.device == nil {
		return "Closed"
	}
	if w.pinwait {
		return fmt.Sprintf("Trezor v%d.%d.%d '%s' waiting for PIN", w.version[0], w.version[1], w.version[2], w.label)
	}
	return fmt.Sprintf("Trezor v%d.%d.%d '%s' online", w.version[0], w.version[1], w.version[2], w.label)
}

// failed returns if the USB device wrapped by the wallet failed for some reason.
// This is used by the device scanner to report failed wallets as departed.
//
// The method assumes that the state lock is *not* held!
func (w *trezorWallet) failed() bool {
	w.stateLock.RLock() // No device communication, state lock is enough
	defer w.stateLock.RUnlock()

	return w.failure != nil
}

// Open implements accounts.Wallet, attempting to open a USB connection to the
// Trezor hardware wallet. Connecting to the Trezor is a two phase operation:
//  * The first phase is to establish the USB connection, initialize it and read
//    the wallet's features. This phase is invoked is the provided passphrase is
//    empty. The device will display the pinpad as a result and will return an
//    appropriate error to notify the user that a second open phase is needed.
//  * The second phase is to unlock access to the Trezor, which is done by the
//    user actually providing a passphrase mapping a keyboard keypad to the pin
//    number of the user (shuffled according to the pinpad displayed).
func (w *trezorWallet) Open(passphrase string) error {
	w.stateLock.Lock() // State lock is enough since there's no connection yet at this point
	defer w.stateLock.Unlock()

	// If phase 1 is requested, init the connection and wait for user callback
	if passphrase == "" {
		// If we're already waiting for a PIN entry, insta-return
		if w.pinwait {
			return ErrTrezorPINNeeded
		}
		// Initialize a connection to the device
		if err := w.openInit(); err != nil {
			return err
		}
		// Do a manual ping, forcing the device to ask for its PIN
		askPin, pinRequest := true, new(trezor.PinMatrixRequest)
		if err := w.trezorExchange(&trezor.Ping{PinProtection: &askPin}, pinRequest); err != nil {
			return err
		}
		w.pinwait = true

		return ErrTrezorPINNeeded
	}
	// Phase 2 requested with actual PIN entry
	w.pinwait = false

	success := new(trezor.Success)
	if err := w.trezorExchange(&trezor.PinMatrixAck{Pin: &passphrase}, success); err != nil {
		w.failure = err
		return err
	}
	go w.hub.updateFeed.Send(accounts.WalletEvent{Wallet: w, Kind: accounts.WalletOpened})

	// Trezor unlocked, start the heartbeat cycle and account derivation
	w.paths = make(map[common.Address]accounts.DerivationPath)

	w.deriveReq = make(chan chan struct{})
	w.deriveQuit = make(chan chan error)
	w.healthQuit = make(chan chan error)

	defer func() {
		go w.heartbeat()
		go w.selfDerive()
	}()
	return nil
}

// openInit is the first phase of a Trezor opening mechanism which initializes
//  device connection and requests the device to display the pinpad.
func (w *trezorWallet) openInit() error {
	// If the wallet was already opened, don't try to phase-1 open again
	if w.device != nil {
		return accounts.ErrWalletAlreadyOpen
	}
	// Otherwise iterate over all USB devices and find this again (no way to directly do this)
	device, err := w.info.Open()
	if err != nil {
		return err
	}
	// Wallet successfully connected to, init the connection and start the heartbeat
	w.device = device
	w.commsLock = make(chan struct{}, 1)
	w.commsLock <- struct{}{} // Enable lock

	// Retrieve the Trezor's version number and user label
	features := new(trezor.Features)
	if err := w.trezorExchange(&trezor.Initialize{}, features); err != nil {
		return err
	}
	w.version = [3]uint32{features.GetMajorVersion(), features.GetMinorVersion(), features.GetPatchVersion()}
	w.label = features.GetLabel()

	return nil
}

// heartbeat is a health check loop for the Trezor wallets to periodically verify
// whether they are still present or if they malfunctioned. It is needed because:
//  - libusb on Windows doesn't support hotplug, so we can't detect USB unplugs
func (w *trezorWallet) heartbeat() {
	w.log.Debug("Trezor health-check started")
	defer w.log.Debug("Trezor health-check stopped")

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
		case <-time.After(heartbeatCycle):
			// Heartbeat time
		}
		// Execute a tiny data exchange to see responsiveness
		w.stateLock.RLock()
		if w.device == nil {
			// Terminated while waiting for the lock
			w.stateLock.RUnlock()
			continue
		}
		<-w.commsLock // Don't lock state while executing ping

		success := new(trezor.Success)
		err = w.trezorExchange(&trezor.Ping{}, success)

		w.commsLock <- struct{}{}
		w.stateLock.RUnlock()

		if err != nil {
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
		w.log.Debug("Trezor health-check failed", "err", err)
		errc = <-w.healthQuit
	}
	errc <- err
}

// Close implements accounts.Wallet, closing the USB connection to the Trezor.
func (w *trezorWallet) Close() error {
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
func (w *trezorWallet) close() error {
	// Allow duplicate closes, especially for health-check failures
	if w.device == nil {
		return nil
	}
	// Close the device, clear everything, then return
	w.device.Close()
	w.device = nil

	w.label, w.version = "", [3]uint32{}
	w.accounts, w.paths = nil, nil

	return nil
}

// Accounts implements accounts.Wallet, returning the list of accounts pinned to
// the Trezor hardware wallet. If self-derivation was enabled, the account list
// is periodically expanded based on current chain state.
func (w *trezorWallet) Accounts() []accounts.Account {
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
func (w *trezorWallet) selfDerive() {
	w.log.Debug("Trezor self-derivation started")
	defer w.log.Debug("Trezor self-derivation stopped")

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
		if w.device == nil || w.deriveChain == nil {
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
				if nextAddr, err = w.trezorDerive(nextPath); err != nil {
					w.log.Warn("Trezor account derivation failed", "err", err)
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
				w.log.Warn("Trezor balance retrieval failed", "err", err)
				break
			}
			nonce, err = w.deriveChain.NonceAt(context, nextAddr, nil)
			if err != nil {
				w.log.Warn("Trezor nonce retrieval failed", "err", err)
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
				w.log.Info("Trezor discovered new account", "address", nextAddr, "path", path, "balance", balance, "nonce", nonce)
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
			case <-time.After(selfDeriveThrottling):
				// Waited enough, willing to self-derive again
			}
		}
	}
	// In case of error, wait for termination
	if err != nil {
		w.log.Debug("Trezor self-derivation failed", "err", err)
		errc = <-w.deriveQuit
	}
	errc <- err
}

// Contains implements accounts.Wallet, returning whether a particular account is
// or is not pinned into this Trezor instance. Although we could attempt to resolve
// unpinned accounts, that would be an non-negligible hardware operation.
func (w *trezorWallet) Contains(account accounts.Account) bool {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	_, exists := w.paths[account.Address]
	return exists
}

// Derive implements accounts.Wallet, deriving a new account at the specific
// derivation path. If pin is set to true, the account will be added to the list
// of tracked accounts.
func (w *trezorWallet) Derive(path accounts.DerivationPath, pin bool) (accounts.Account, error) {
	// Try to derive the actual account and update its URL if successful
	w.stateLock.RLock() // Avoid device disappearing during derivation

	if w.device == nil {
		w.stateLock.RUnlock()
		return accounts.Account{}, accounts.ErrWalletClosed
	}
	<-w.commsLock // Avoid concurrent hardware access
	address, err := w.trezorDerive(path)
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
func (w *trezorWallet) SelfDerive(base accounts.DerivationPath, chain ethereum.ChainStateReader) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	w.deriveNextPath = make(accounts.DerivationPath, len(base))
	copy(w.deriveNextPath[:], base[:])

	w.deriveNextAddr = common.Address{}
	w.deriveChain = chain
}

// SignHash implements accounts.Wallet, however signing arbitrary data is not
// supported for Trezor wallets, so this method will always return an error.
func (w *trezorWallet) SignHash(acc accounts.Account, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTx implements accounts.Wallet. It sends the transaction over to the Trezor
// wallet to request a confirmation from the user. It returns either the signed
// transaction or a failure if the user denied the transaction.
func (w *trezorWallet) SignTx(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	w.stateLock.RLock() // Comms have own mutex, this is for the state fields
	defer w.stateLock.RUnlock()

	// If the wallet is closed, abort
	if w.device == nil {
		return nil, accounts.ErrWalletClosed
	}
	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}
	// All infos gathered and metadata checks out, request signing
	<-w.commsLock
	defer func() { w.commsLock <- struct{}{} }()

	// Ensure the device isn't screwed with while user confirmation is pending
	// TODO(karalabe): remove if hotplug lands on Windows
	w.hub.commsLock.Lock()
	w.hub.commsPend++
	w.hub.commsLock.Unlock()

	defer func() {
		w.hub.commsLock.Lock()
		w.hub.commsPend--
		w.hub.commsLock.Unlock()
	}()

	return w.trezorSign(path, account.Address, tx, chainID)
}

// SignHashWithPassphrase implements accounts.Wallet, however signing arbitrary
// data is not supported for Trezor wallets, so this method will always return
// an error.
func (w *trezorWallet) SignHashWithPassphrase(account accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTxWithPassphrase implements accounts.Wallet, attempting to sign the given
// transaction with the given account using passphrase as extra authentication.
// Since the Trezor does not support extra passphrases, it is silently ignored.
func (w *trezorWallet) SignTxWithPassphrase(account accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return w.SignTx(account, tx, chainID)
}

// trezorDerive sends a derivation request to the Trezor device and returns the
// Ethereum address located on that path.
func (w *trezorWallet) trezorDerive(derivationPath []uint32) (common.Address, error) {
	address := new(trezor.EthereumAddress)
	if err := w.trezorExchange(&trezor.EthereumGetAddress{AddressN: derivationPath}, address); err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(address.GetAddress()), nil
}

// trezorSign sends the transaction to the Trezor wallet, and waits for the user
// to confirm or deny the transaction.
func (w *trezorWallet) trezorSign(derivationPath []uint32, address common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// Create the transaction initiation message
	data := tx.Data()
	length := uint32(len(data))

	request := &trezor.EthereumSignTx{
		AddressN:   derivationPath,
		Nonce:      new(big.Int).SetUint64(tx.Nonce()).Bytes(),
		GasPrice:   tx.GasPrice().Bytes(),
		GasLimit:   tx.Gas().Bytes(),
		Value:      tx.Value().Bytes(),
		DataLength: &length,
	}
	if to := tx.To(); to != nil {
		request.To = (*to)[:] // Non contract deploy, set recipient explicitly
	}
	if length > 1024 { // Send the data chunked if that was requested
		request.DataInitialChunk, data = data[:1024], data[1024:]
	} else {
		request.DataInitialChunk, data = data, nil
	}
	if chainID != nil { // EIP-155 transaction, set chain ID explicitly (only 32 bit is supported!?)
		id := uint32(chainID.Int64())
		request.ChainId = &id
	}
	// Send the initiation message and stream content until a signature is returned
	response := new(trezor.EthereumTxRequest)
	if err := w.trezorExchange(request, response); err != nil {
		return nil, err
	}
	for response.DataLength != nil && int(*response.DataLength) <= len(data) {
		chunk := data[:*response.DataLength]
		data = data[*response.DataLength:]

		if err := w.trezorExchange(&trezor.EthereumTxAck{DataChunk: chunk}, response); err != nil {
			return nil, err
		}
	}
	// Extract the Ethereum signature and do a sanity validation
	if len(response.GetSignatureR()) == 0 || len(response.GetSignatureS()) == 0 || response.GetSignatureV() == 0 {
		return nil, errors.New("reply lacks signature")
	}
	signature := append(append(response.GetSignatureR(), response.GetSignatureS()...), byte(response.GetSignatureV()))

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

// trezorExchange performs a data exchange with the Trezor wallet, sending it a
// message and retrieving the response.
func (w *trezorWallet) trezorExchange(req proto.Message, res proto.Message) error {
	// Construct the original message payload to chunk up
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	payload := make([]byte, 8+len(data))
	copy(payload, []byte{0x23, 0x23})
	binary.BigEndian.PutUint16(payload[2:], trezor.Type(req))
	binary.BigEndian.PutUint32(payload[4:], uint32(len(data)))
	copy(payload[8:], data)

	// Stream all the chunks to the device
	chunk := make([]byte, 64)
	chunk[0] = 0x3f // Report ID magic number

	for len(payload) > 0 {
		// Construct the new message to stream, padding with zeroes if needed
		if len(payload) > 63 {
			copy(chunk[1:], payload[:63])
			payload = payload[63:]
		} else {
			copy(chunk[1:], payload)
			copy(chunk[1+len(payload):], make([]byte, 63-len(payload)))
			payload = nil
		}
		// Send over to the device
		w.log.Trace("Data chunk sent to the Trezor", "chunk", hexutil.Bytes(chunk))
		if _, err := w.device.Write(chunk); err != nil {
			return err
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var (
		kind  uint16
		reply []byte
	)
	for {
		// Read the next chunk from the Trezor wallet
		if _, err := io.ReadFull(w.device, chunk); err != nil {
			return err
		}
		w.log.Trace("Data chunk received from the Trezor", "chunk", hexutil.Bytes(chunk))

		// Make sure the transport header matches
		if chunk[0] != 0x3f || (len(reply) == 0 && (chunk[1] != 0x23 || chunk[2] != 0x23)) {
			return errReplyInvalidHeader
		}
		// If it's the first chunk, retrieve the reply message type and total message length
		var payload []byte

		if len(reply) == 0 {
			kind = binary.BigEndian.Uint16(chunk[3:5])
			reply = make([]byte, 0, int(binary.BigEndian.Uint32(chunk[5:9])))
			payload = chunk[9:]
		} else {
			payload = chunk[1:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}
	}
	// Try to parse the reply into the requested reply message
	if kind == uint16(trezor.MessageType_MessageType_Failure) {
		// Trezor returned a failure, extract and return the message
		failure := new(trezor.Failure)
		if err := proto.Unmarshal(reply, failure); err != nil {
			return err
		}
		return errors.New("trezor: " + failure.GetMessage())
	}
	if kind == uint16(trezor.MessageType_MessageType_ButtonRequest) {
		// Trezor is waitinf for user confirmation, ack and wait for the next message
		return w.trezorExchange(&trezor.ButtonAck{}, res)
	}
	if want := trezor.Type(res); kind != want {
		return fmt.Errorf("trezor: expected reply type %s, got %s", trezor.Name(want), trezor.Name(kind))
	}
	return proto.Unmarshal(reply, res)
}
