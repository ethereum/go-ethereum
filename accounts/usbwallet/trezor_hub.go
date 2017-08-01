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

package usbwallet

import (
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/karalabe/hid"
)

// TrezorScheme is the protocol scheme prefixing account and wallet URLs.
var TrezorScheme = "trezor"

// trezorVendorID is the USB vendor ID for SatoshiLabs.
var trezorVendorID = uint16(0x534c)

// trezorDeviceID is the USB device ID for the Trezor 1.
var trezorDeviceID = uint16(0x0001)

// Maximum time between wallet refreshes (if USB hotplug notifications don't work).
const trezorRefreshCycle = time.Second

// Minimum time between wallet refreshes to avoid USB trashing.
const trezorRefreshThrottling = 500 * time.Millisecond

// TrezorHub is a accounts.Backend that can find and handle Trezor hardware wallets.
type TrezorHub struct {
	refreshed   time.Time               // Time instance when the list of wallets was last refreshed
	wallets     []accounts.Wallet       // List of Trezor devices currently tracking
	updateFeed  event.Feed              // Event feed to notify wallet additions/removals
	updateScope event.SubscriptionScope // Subscription scope tracking current live listeners
	updating    bool                    // Whether the event notification loop is running

	quit chan chan error

	stateLock sync.RWMutex // Protects the internals of the hub from racey access

	// TODO(karalabe): remove if hotplug lands on Windows
	commsPend int        // Number of operations blocking enumeration
	commsLock sync.Mutex // Lock protecting the pending counter and enumeration
}

// NewTrezorHub creates a new hardware wallet manager for Trezor devices.
func NewTrezorHub() (*TrezorHub, error) {
	if !hid.Supported() {
		return nil, errors.New("unsupported platform")
	}
	hub := &TrezorHub{
		quit: make(chan chan error),
	}
	hub.refreshWallets()
	return hub, nil
}

// Wallets implements accounts.Backend, returning all the currently tracked USB
// devices that appear to be Trezor hardware wallets.
func (hub *TrezorHub) Wallets() []accounts.Wallet {
	// Make sure the list of wallets is up to date
	hub.refreshWallets()

	hub.stateLock.RLock()
	defer hub.stateLock.RUnlock()

	cpy := make([]accounts.Wallet, len(hub.wallets))
	copy(cpy, hub.wallets)
	return cpy
}

// refreshWallets scans the USB devices attached to the machine and updates the
// list of wallets based on the found devices.
func (hub *TrezorHub) refreshWallets() {
	// Don't scan the USB like crazy it the user fetches wallets in a loop
	hub.stateLock.RLock()
	elapsed := time.Since(hub.refreshed)
	hub.stateLock.RUnlock()

	if elapsed < trezorRefreshThrottling {
		return
	}
	// Retrieve the current list of Trezor devices
	var trezors []hid.DeviceInfo

	if runtime.GOOS == "linux" {
		// hidapi on Linux opens the device during enumeration to retrieve some infos,
		// breaking the Trezor protocol if that is waiting for user confirmation. This
		// is a bug acknowledged at Trezor, but it won't be fixed on old devices so we
		// need to prevent concurrent comms ourselves. The more elegant solution would
		// be to ditch enumeration in favor of hutplug events, but that don't work yet
		// on Windows so if we need to hack it anyway, this is more elegant for now.
		hub.commsLock.Lock()
		if hub.commsPend > 0 { // A confirmation is pending, don't refresh
			hub.commsLock.Unlock()
			return
		}
	}
	for _, info := range hid.Enumerate(trezorVendorID, trezorDeviceID) {
		if info.Interface == 0 { // interface #1 is the debug link, skip it
			trezors = append(trezors, info)
		}
	}
	if runtime.GOOS == "linux" {
		// See rationale before the enumeration why this is needed and only on Linux.
		hub.commsLock.Unlock()
	}
	// Transform the current list of wallets into the new one
	hub.stateLock.Lock()

	wallets := make([]accounts.Wallet, 0, len(trezors))
	events := []accounts.WalletEvent{}

	for _, trezor := range trezors {
		url := accounts.URL{Scheme: TrezorScheme, Path: trezor.Path}

		// Drop wallets in front of the next device or those that failed for some reason
		for len(hub.wallets) > 0 && (hub.wallets[0].URL().Cmp(url) < 0 || hub.wallets[0].(*trezorWallet).failed()) {
			events = append(events, accounts.WalletEvent{Wallet: hub.wallets[0], Kind: accounts.WalletDropped})
			hub.wallets = hub.wallets[1:]
		}
		// If there are no more wallets or the device is before the next, wrap new wallet
		if len(hub.wallets) == 0 || hub.wallets[0].URL().Cmp(url) > 0 {
			wallet := &trezorWallet{hub: hub, url: &url, info: trezor, log: log.New("url", url)}

			events = append(events, accounts.WalletEvent{Wallet: wallet, Kind: accounts.WalletArrived})
			wallets = append(wallets, wallet)
			continue
		}
		// If the device is the same as the first wallet, keep it
		if hub.wallets[0].URL().Cmp(url) == 0 {
			wallets = append(wallets, hub.wallets[0])
			hub.wallets = hub.wallets[1:]
			continue
		}
	}
	// Drop any leftover wallets and set the new batch
	for _, wallet := range hub.wallets {
		events = append(events, accounts.WalletEvent{Wallet: wallet, Kind: accounts.WalletDropped})
	}
	hub.refreshed = time.Now()
	hub.wallets = wallets
	hub.stateLock.Unlock()

	// Fire all wallet events and return
	for _, event := range events {
		hub.updateFeed.Send(event)
	}
}

// Subscribe implements accounts.Backend, creating an async subscription to
// receive notifications on the addition or removal of Trezor wallets.
func (hub *TrezorHub) Subscribe(sink chan<- accounts.WalletEvent) event.Subscription {
	// We need the mutex to reliably start/stop the update loop
	hub.stateLock.Lock()
	defer hub.stateLock.Unlock()

	// Subscribe the caller and track the subscriber count
	sub := hub.updateScope.Track(hub.updateFeed.Subscribe(sink))

	// Subscribers require an active notification loop, start it
	if !hub.updating {
		hub.updating = true
		go hub.updater()
	}
	return sub
}

// updater is responsible for maintaining an up-to-date list of wallets stored in
// the keystore, and for firing wallet addition/removal events. It listens for
// account change events from the underlying account cache, and also periodically
// forces a manual refresh (only triggers for systems where the filesystem notifier
// is not running).
func (hub *TrezorHub) updater() {
	for {
		// Wait for a USB hotplug event (not supported yet) or a refresh timeout
		select {
		//case <-hub.changes: // reenable on hutplug implementation
		case <-time.After(trezorRefreshCycle):
		}
		// Run the wallet refresher
		hub.refreshWallets()

		// If all our subscribers left, stop the updater
		hub.stateLock.Lock()
		if hub.updateScope.Count() == 0 {
			hub.updating = false
			hub.stateLock.Unlock()
			return
		}
		hub.stateLock.Unlock()
	}
}
