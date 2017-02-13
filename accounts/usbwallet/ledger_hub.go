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
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/event"
	"github.com/karalabe/gousb/usb"
)

// LedgerScheme is the protocol scheme prefixing account and wallet URLs.
var LedgerScheme = "ledger"

// ledgerDeviceIDs are the known device IDs that Ledger wallets use.
var ledgerDeviceIDs = []deviceID{
	{Vendor: 0x2c97, Product: 0x0000}, // Ledger Blue
	{Vendor: 0x2c97, Product: 0x0001}, // Ledger Nano S
}

// Maximum time between wallet refreshes (if USB hotplug notifications don't work).
const ledgerRefreshCycle = time.Second

// Minimum time between wallet refreshes to avoid USB trashing.
const ledgerRefreshThrottling = 500 * time.Millisecond

// LedgerHub is a accounts.Backend that can find and handle Ledger hardware wallets.
type LedgerHub struct {
	ctx *usb.Context // Context interfacing with a libusb instance

	refreshed   time.Time               // Time instance when the list of wallets was last refreshed
	wallets     []accounts.Wallet       // List of Ledger devices currently tracking
	updateFeed  event.Feed              // Event feed to notify wallet additions/removals
	updateScope event.SubscriptionScope // Subscription scope tracking current live listeners
	updating    bool                    // Whether the event notification loop is running

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
		ctx:  ctx,
		quit: make(chan chan error),
	}
	hub.refreshWallets()

	return hub, nil
}

// Wallets implements accounts.Backend, returning all the currently tracked USB
// devices that appear to be Ledger hardware wallets.
func (hub *LedgerHub) Wallets() []accounts.Wallet {
	// Make sure the list of wallets is up to date
	hub.refreshWallets()

	hub.lock.RLock()
	defer hub.lock.RUnlock()

	cpy := make([]accounts.Wallet, len(hub.wallets))
	copy(cpy, hub.wallets)
	return cpy
}

// refreshWallets scans the USB devices attached to the machine and updates the
// list of wallets based on the found devices.
func (hub *LedgerHub) refreshWallets() {
	// Don't scan the USB like crazy it the user fetches wallets in a loop
	hub.lock.RLock()
	elapsed := time.Since(hub.refreshed)
	hub.lock.RUnlock()

	if elapsed < ledgerRefreshThrottling {
		return
	}
	// Retrieve the current list of Ledger devices
	var devIDs []deviceID
	var busIDs []uint16

	hub.ctx.ListDevices(func(desc *usb.Descriptor) bool {
		// Gather Ledger devices, don't connect any just yet
		for _, id := range ledgerDeviceIDs {
			if desc.Vendor == id.Vendor && desc.Product == id.Product {
				devIDs = append(devIDs, deviceID{Vendor: desc.Vendor, Product: desc.Product})
				busIDs = append(busIDs, uint16(desc.Bus)<<8+uint16(desc.Address))
				return false
			}
		}
		// Not ledger, ignore and don't connect either
		return false
	})
	// Transform the current list of wallets into the new one
	hub.lock.Lock()

	wallets := make([]accounts.Wallet, 0, len(devIDs))
	events := []accounts.WalletEvent{}

	for i := 0; i < len(devIDs); i++ {
		devID, busID := devIDs[i], busIDs[i]

		url := accounts.URL{Scheme: LedgerScheme, Path: fmt.Sprintf("%03d:%03d", busID>>8, busID&0xff)}

		// Drop wallets in front of the next device or those that failed for some reason
		for len(hub.wallets) > 0 && (hub.wallets[0].URL().Cmp(url) < 0 || hub.wallets[0].(*ledgerWallet).failed()) {
			events = append(events, accounts.WalletEvent{Wallet: hub.wallets[0], Arrive: false})
			hub.wallets = hub.wallets[1:]
		}
		// If there are no more wallets or the device is before the next, wrap new wallet
		if len(hub.wallets) == 0 || hub.wallets[0].URL().Cmp(url) > 0 {
			wallet := &ledgerWallet{context: hub.ctx, hardwareID: devID, locationID: busID, url: &url}

			events = append(events, accounts.WalletEvent{Wallet: wallet, Arrive: true})
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
		events = append(events, accounts.WalletEvent{Wallet: wallet, Arrive: false})
	}
	hub.refreshed = time.Now()
	hub.wallets = wallets
	hub.lock.Unlock()

	// Fire all wallet events and return
	for _, event := range events {
		hub.updateFeed.Send(event)
	}
}

// Subscribe implements accounts.Backend, creating an async subscription to
// receive notifications on the addition or removal of Ledger wallets.
func (hub *LedgerHub) Subscribe(sink chan<- accounts.WalletEvent) event.Subscription {
	// We need the mutex to reliably start/stop the update loop
	hub.lock.Lock()
	defer hub.lock.Unlock()

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
func (hub *LedgerHub) updater() {
	for {
		// Wait for a USB hotplug event (not supported yet) or a refresh timeout
		select {
		//case <-hub.changes: // reenable on hutplug implementation
		case <-time.After(ledgerRefreshCycle):
		}
		// Run the wallet refresher
		hub.refreshWallets()

		// If all our subscribers left, stop the updater
		hub.lock.Lock()
		if hub.updateScope.Count() == 0 {
			hub.updating = false
			hub.lock.Unlock()
			return
		}
		hub.lock.Unlock()
	}
}
