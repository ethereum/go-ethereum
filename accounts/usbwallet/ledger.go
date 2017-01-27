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

// +build !ios

package usbwallet

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/karalabe/gousb/usb"
)

// ledgerDeviceIDs are the known device IDs that Ledger wallets use.
var ledgerDeviceIDs = []deviceID{
	{Vendor: 0x2c97, Product: 0x0000}, // Ledger Blue
	{Vendor: 0x2c97, Product: 0x0001}, // Ledger Nano S
}

// ledgerDerivationPath is the key derivation parameters used by the wallet.
var ledgerDerivationPath = [4]uint32{0x80000000 + 44, 0x80000000 + 60, 0x80000000 + 0, 0}

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

// ledgerWallet represents a live USB Ledger hardware wallet.
type ledgerWallet struct {
	device *usb.Device  // USB device advertising itself as a Ledger wallet
	input  usb.Endpoint // Input endpoint to send data to this device
	output usb.Endpoint // Output endpoint to receive data from this device

	address common.Address // Current address of the wallet (may be zero if Ethereum app offline)
	url     string         // Textual URL uniquely identifying this wallet
	version [3]byte        // Current version of the Ledger Ethereum app (zero if app is offline)
}

// LedgerHub is a USB hardware wallet interface that can find and handle Ledger
// wallets.
type LedgerHub struct {
	ctx *usb.Context // Context interfacing with a libusb instance

	wallets  map[uint16]*ledgerWallet  // Apparent Ledger wallets (some may be inactive)
	accounts []accounts.Account        // List of active Ledger accounts
	index    map[common.Address]uint16 // Set of addresses with active wallets

	quit chan chan error
	lock sync.RWMutex
}

// NewLedgerHub creates a new hardware wallet manager for Ledger devices.
func NewLedgerHub() (*LedgerHub, error) {
	// Initialize the USB library to access Ledgers through
	ctx, err := usb.NewContext()
	if err != nil {
		return nil, err
	}
	// Create the USB hub, start and return it
	hub := &LedgerHub{
		ctx:     ctx,
		wallets: make(map[uint16]*ledgerWallet),
		index:   make(map[common.Address]uint16),
		quit:    make(chan chan error),
	}
	go hub.watch()
	return hub, nil
}

// Accounts retrieves the live of accounts currently known by the Ledger hub.
func (hub *LedgerHub) Accounts() []accounts.Account {
	hub.lock.RLock()
	defer hub.lock.RUnlock()

	cpy := make([]accounts.Account, len(hub.accounts))
	copy(cpy, hub.accounts)
	return cpy
}

// HasAddress reports whether an account with the given address is present.
func (hub *LedgerHub) HasAddress(addr common.Address) bool {
	hub.lock.RLock()
	defer hub.lock.RUnlock()

	_, known := hub.index[addr]
	return known
}

// SignHash is not supported for Ledger wallets, so this method will always
// return an error.
func (hub *LedgerHub) SignHash(acc accounts.Account, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTx sends the transaction over to the Ledger wallet to request a confirmation
// from the user. It returns either the signed transaction or a failure if the user
// denied the transaction.
//
// Note, if the version of the Ethereum application running on the Ledger wallet is
// too old to sign EIP-155 transactions, but such is requested nonetheless, an error
// will be returned opposed to silently signing in Homestead mode.
func (hub *LedgerHub) SignTx(acc accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	hub.lock.RLock()
	defer hub.lock.RUnlock()

	// If the account contains the device URL, flatten it to make sure
	var id uint16
	if acc.URL != "" {
		if parts := strings.Split(acc.URL, "."); len(parts) == 2 {
			bus, busErr := strconv.Atoi(parts[0])
			addr, addrErr := strconv.Atoi(parts[1])

			if busErr == nil && addrErr == nil {
				id = uint16(bus)<<8 + uint16(addr)
			}
		}
	}
	// If the id is still zero, URL is either missing or bad, resolve
	if id == 0 {
		var ok bool
		if id, ok = hub.index[acc.Address]; !ok {
			return nil, accounts.ErrUnknownAccount
		}
	}
	// Retrieve the wallet associated with the URL
	wallet, ok := hub.wallets[id]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}
	// Ensure the wallet is capable of signing the given transaction
	if chainID != nil && wallet.version[0] <= 1 && wallet.version[1] <= 0 && wallet.version[2] <= 2 {
		return nil, fmt.Errorf("Ledger v%d.%d.%d doesn't support signing this transaction, please update to v1.0.3 at least",
			wallet.version[0], wallet.version[1], wallet.version[2])
	}
	return wallet.sign(tx, chainID)
}

// SignHashWithPassphrase is not supported for Ledger wallets, so this method
// will always return an error.
func (hub *LedgerHub) SignHashWithPassphrase(acc accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	return nil, accounts.ErrNotSupported
}

// SignTxWithPassphrase requests the backend to sign the given transaction, with the
// given passphrase as extra authentication information. Since the Ledger does not
// support this feature, it will just silently ignore the passphrase.
func (hub *LedgerHub) SignTxWithPassphrase(acc accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return hub.SignTx(acc, tx, chainID)
}

// Close terminates the usb watching for Ledger wallets and returns when it
// successfully terminated.
func (hub *LedgerHub) Close() error {
	// Terminate the USB scanner
	errc := make(chan error)
	hub.quit <- errc
	err := <-errc

	// Release the USB interface and return
	hub.ctx.Close()
	return err
}

// watch starts watching the local machine's USB ports for the connection or
// disconnection of Ledger devices.
func (hub *LedgerHub) watch() {
	for {
		// Rescan the USB ports for devices newly added or removed
		hub.rescan()

		// Sleep for a certain amount of time or until terminated
		select {
		case errc := <-hub.quit:
			errc <- nil
			return
		case <-time.After(time.Second):
		}
	}
}

// rescan searches the USB ports for attached Ledger hardware wallets.
func (hub *LedgerHub) rescan() {
	hub.lock.Lock()
	defer hub.lock.Unlock()

	// Iterate over all attached devices and fetch those seemingly Ledger
	present := make(map[uint16]bool)
	devices, _ := hub.ctx.ListDevices(func(desc *usb.Descriptor) bool {
		// Discard all devices not advertizing as Ledger
		ledger := false
		for _, id := range ledgerDeviceIDs {
			if desc.Vendor == id.Vendor && desc.Product == id.Product {
				ledger = true
			}
		}
		if !ledger {
			return false
		}
		// If we have a Ledger, mark as still present, or open as new
		id := uint16(desc.Bus)<<8 + uint16(desc.Address)
		if _, known := hub.wallets[id]; known {
			// Track it's presence, but don't open again
			present[id] = true
			return false
		}
		// New Ledger device, open it for communication
		return true
	})
	// Drop any tracker wallet which disconnected
	for id, wallet := range hub.wallets {
		if !present[id] {
			if wallet.address == (common.Address{}) {
				glog.V(logger.Info).Infof("ledger wallet [%03d.%03d] disconnected", wallet.device.Bus, wallet.device.Address)
			} else {
				// A live account disconnected, remove it from the tracked accounts
				for i, account := range hub.accounts {
					if account.Address == wallet.address && account.URL == wallet.url {
						hub.accounts = append(hub.accounts[:i], hub.accounts[i+1:]...)
						break
					}
				}
				delete(hub.index, wallet.address)

				glog.V(logger.Info).Infof("ledger wallet [%03d.%03d] v%d.%d.%d disconnected: %s", wallet.device.Bus, wallet.device.Address,
					wallet.version[0], wallet.version[1], wallet.version[2], wallet.address.Hex())
			}
			delete(hub.wallets, id)
			wallet.device.Close()
		}
	}
	// Start tracking all wallets which newly appeared
	var err error
	for _, device := range devices {
		// Make sure the alleged device has the correct IO endpoints
		wallet := &ledgerWallet{
			device: device,
			url:    fmt.Sprintf("%03d.%03d", device.Bus, device.Address),
		}
		var invalid string
		switch {
		case len(device.Descriptor.Configs) == 0:
			invalid = "no endpoint config available"
		case len(device.Descriptor.Configs[0].Interfaces) == 0:
			invalid = "no endpoint interface available"
		case len(device.Descriptor.Configs[0].Interfaces[0].Setups) == 0:
			invalid = "no endpoint setup available"
		case len(device.Descriptor.Configs[0].Interfaces[0].Setups[0].Endpoints) < 2:
			invalid = "not enough IO endpoints available"
		}
		if invalid != "" {
			glog.V(logger.Debug).Infof("ledger wallet [%s] deemed invalid: %s", wallet.url, invalid)
			device.Close()
			continue
		}
		// Open the input and output endpoints to the device
		wallet.input, err = device.OpenEndpoint(
			device.Descriptor.Configs[0].Config,
			device.Descriptor.Configs[0].Interfaces[0].Number,
			device.Descriptor.Configs[0].Interfaces[0].Setups[0].Number,
			device.Descriptor.Configs[0].Interfaces[0].Setups[0].Endpoints[1].Address,
		)
		if err != nil {
			glog.V(logger.Debug).Infof("ledger wallet [%s] input open failed: %v", wallet.url, err)
			device.Close()
			continue
		}
		wallet.output, err = device.OpenEndpoint(
			device.Descriptor.Configs[0].Config,
			device.Descriptor.Configs[0].Interfaces[0].Number,
			device.Descriptor.Configs[0].Interfaces[0].Setups[0].Number,
			device.Descriptor.Configs[0].Interfaces[0].Setups[0].Endpoints[0].Address,
		)
		if err != nil {
			glog.V(logger.Debug).Infof("ledger wallet [%s] output open failed: %v", wallet.url, err)
			device.Close()
			continue
		}
		// Start tracking the device as a probably Ledger wallet
		id := uint16(device.Bus)<<8 + uint16(device.Address)
		hub.wallets[id] = wallet

		if wallet.resolveAddress() != nil {
			glog.V(logger.Info).Infof("ledger wallet [%s] connected, Ethereum app not started", wallet.url)
		} else {
			// Try to resolve the Ethereum app's version, will fail prior to v1.0.2
			if wallet.resolveVersion() != nil {
				wallet.version = [3]byte{1, 0, 0} // Assume worst case, can't verify if v1.0.0 or v1.0.1
			}
			hub.accounts = append(hub.accounts, accounts.Account{
				Address: wallet.address,
				URL:     wallet.url,
			})
			hub.index[wallet.address] = id

			glog.V(logger.Info).Infof("ledger wallet [%s] v%d.%d.%d connected: %s", wallet.url,
				wallet.version[0], wallet.version[1], wallet.version[2], wallet.address.Hex())
		}
	}
}

// resolveVersion retrieves the current version of the Ethereum wallet app running
// on the Ledger wallet and caches it for future reference.
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
func (wallet *ledgerWallet) resolveVersion() error {
	// Send the request and wait for the response
	reply, err := wallet.exchange(ledgerOpGetConfiguration, 0, 0, nil)
	if err != nil {
		return err
	}
	if len(reply) != 4 {
		return errors.New("reply not of correct size")
	}
	// Cache the version for future reference
	copy(wallet.version[:], reply[1:])
	return nil
}

// resolveAddress retrieves the currently active Ethereum address from a Ledger
// wallet and caches it for future reference.
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
func (wallet *ledgerWallet) resolveAddress() error {
	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(ledgerDerivationPath))
	path[0] = byte(len(ledgerDerivationPath))
	for i, component := range ledgerDerivationPath {
		binary.BigEndian.PutUint32(path[1+4*i:], component)
	}
	// Send the request and wait for the response
	reply, err := wallet.exchange(ledgerOpRetrieveAddress, ledgerP1DirectlyFetchAddress, ledgerP2DiscardAddressChainCode, path)
	if err != nil {
		return err
	}
	// Discard the public key, we don't need that for now
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return errors.New("reply lacks public key entry")
	}
	reply = reply[1+int(reply[0]):]

	// Extract the Ethereum hex address string
	if len(reply) < 1 || len(reply) < 1+int(reply[0]) {
		return errors.New("reply lacks address entry")
	}
	hexstr := reply[1 : 1+int(reply[0])]

	// Decode the hex sting into an Ethereum address and return
	hex.Decode(wallet.address[:], hexstr)
	return nil
}

// sign sends the transaction to the Ledger wallet, and waits for the user to
// confirm or deny the transaction.
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
func (wallet *ledgerWallet) sign(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// We need to modify the timeouts to account for user feedback
	defer func(old time.Duration) { wallet.device.ReadTimeout = old }(wallet.device.ReadTimeout)
	wallet.device.ReadTimeout = time.Minute

	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(ledgerDerivationPath))
	path[0] = byte(len(ledgerDerivationPath))
	for i, component := range ledgerDerivationPath {
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
		reply, err = wallet.exchange(ledgerOpSignTransaction, op, 0, payload[:chunk])
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
		signature[64] = (signature[64]-34)/2 - byte(chainID.Uint64())
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
	if sender != wallet.address {
		glog.V(logger.Error).Infof("Ledger signer mismatch: expected %s, got %s", wallet.address.Hex(), sender.Hex())
		return nil, fmt.Errorf("signer mismatch: expected %s, got %s", wallet.address.Hex(), sender.Hex())
	}
	return signed, nil
}

// exchange performs a data exchange with the Ledger wallet, sending it a message
// and retrieving the response.
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
func (wallet *ledgerWallet) exchange(opcode ledgerOpcode, p1 ledgerParam1, p2 ledgerParam2, data []byte) ([]byte, error) {
	// Construct the message payload, possibly split into multiple chunks
	var chunks [][]byte
	for left := data; len(left) > 0 || len(chunks) == 0; {
		// Create the chunk header
		var chunk []byte

		if len(chunks) == 0 {
			// The first chunk encodes the length and all the opcodes
			chunk = []byte{0x00, 0x00, 0xe0, byte(opcode), byte(p1), byte(p2), byte(len(data))}
			binary.BigEndian.PutUint16(chunk, uint16(5+len(data)))
		}
		// Append the data blob to the end of the chunk
		space := 64 - len(chunk) - 5 // 5 == header size
		if len(left) > space {
			chunks, left = append(chunks, append(chunk, left[:space]...)), left[space:]
			continue
		}
		chunks, left = append(chunks, append(chunk, left...)), nil
	}
	// Stream all the chunks to the device
	for i, chunk := range chunks {
		// Construct the new message to stream
		header := []byte{0x01, 0x01, 0x05, 0x00, 0x00} // Channel ID and command tag appended
		binary.BigEndian.PutUint16(header[3:], uint16(i))

		msg := append(header, chunk...)

		// Send over to the device
		if glog.V(logger.Core) {
			glog.Infof("-> %03d.%03d: %x", wallet.device.Bus, wallet.device.Address, msg)
		}
		if _, err := wallet.input.Write(msg); err != nil {
			return nil, err
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var reply []byte
	for {
		// Read the next chunk from the Ledger wallet
		chunk := make([]byte, 64)
		if _, err := io.ReadFull(wallet.output, chunk); err != nil {
			return nil, err
		}
		if glog.V(logger.Core) {
			glog.Infof("<- %03d.%03d: %x", wallet.device.Bus, wallet.device.Address, chunk)
		}
		// Make sure the transport header matches
		if chunk[0] != 0x01 || chunk[1] != 0x01 || chunk[2] != 0x05 {
			return nil, fmt.Errorf("invalid reply header: %x", chunk[:3])
		}
		// If it's the first chunk, retrieve the total message length
		if chunk[3] == 0x00 && chunk[4] == 0x00 {
			reply = make([]byte, 0, int(binary.BigEndian.Uint16(chunk[5:7])))
			chunk = chunk[7:]
		} else {
			chunk = chunk[5:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(chunk) {
			reply = append(reply, chunk...)
		} else {
			reply = append(reply, chunk[:left]...)
			break
		}
	}
	return reply[:len(reply)-2], nil
}
