// Copyright 2018 The go-ethereum Authors
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

package scwallet

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/ebfe/scard"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

const Scheme = "pcsc"

// refreshCycle is the maximum time between wallet refreshes (if USB hotplug
// notifications don't work).
const refreshCycle = 5 * time.Second

// refreshThrottling is the minimum time between wallet refreshes to avoid thrashing.
const refreshThrottling = 500 * time.Millisecond

// SmartcardPairing contains information about a smart card we have paired with
// or might pair withub.
type SmartcardPairing struct {
	PublicKey    []byte                                     `json:"publicKey"`
	PairingIndex uint8                                      `json:"pairingIndex"`
	PairingKey   []byte                                     `json:"pairingKey"`
	Accounts     map[common.Address]accounts.DerivationPath `json:"accounts"`
}

// Hub is a accounts.Backend that can find and handle generic PC/SC hardware wallets.
type Hub struct {
	scheme string // Protocol scheme prefixing account and wallet URLs.

	context     *scard.Context
	datadir     string
	pairings    map[string]SmartcardPairing
	refreshed   time.Time               // Time instance when the list of wallets was last refreshed
	wallets     map[string]*Wallet      // Mapping from reader names to wallet instances
	updateFeed  event.Feed              // Event feed to notify wallet additions/removals
	updateScope event.SubscriptionScope // Subscription scope tracking current live listeners
	updating    bool                    // Whether the event notification loop is running

	quit chan chan error

	stateLock sync.Mutex // Protects the internals of the hub from racey access
}

var HubType = reflect.TypeOf(&Hub{})

func (hub *Hub) readPairings() error {
	hub.pairings = make(map[string]SmartcardPairing)
	pairingFile, err := os.Open(hub.datadir + "/smartcards.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	pairingData, err := ioutil.ReadAll(pairingFile)
	if err != nil {
		return err
	}
	var pairings []SmartcardPairing
	if err := json.Unmarshal(pairingData, &pairings); err != nil {
		return err
	}

	for _, pairing := range pairings {
		hub.pairings[string(pairing.PublicKey)] = pairing
	}
	return nil
}

func (hub *Hub) writePairings() error {
	pairingFile, err := os.OpenFile(hub.datadir+"/smartcards.json", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	pairings := make([]SmartcardPairing, 0, len(hub.pairings))
	for _, pairing := range hub.pairings {
		pairings = append(pairings, pairing)
	}

	pairingData, err := json.Marshal(pairings)
	if err != nil {
		return err
	}

	if _, err := pairingFile.Write(pairingData); err != nil {
		return err
	}

	return pairingFile.Close()
}

func (hub *Hub) getPairing(wallet *Wallet) *SmartcardPairing {
	pairing, ok := hub.pairings[string(wallet.PublicKey)]
	if ok {
		return &pairing
	}
	return nil
}

func (hub *Hub) setPairing(wallet *Wallet, pairing *SmartcardPairing) error {
	if pairing == nil {
		delete(hub.pairings, string(wallet.PublicKey))
	} else {
		hub.pairings[string(wallet.PublicKey)] = *pairing
	}
	return hub.writePairings()
}

// NewHub creates a new hardware wallet manager for smartcards.
func NewHub(scheme string, datadir string) (*Hub, error) {
	context, err := scard.EstablishContext()
	if err != nil {
		return nil, err
	}

	hub := &Hub{
		scheme:  scheme,
		context: context,
		datadir: datadir,
		wallets: make(map[string]*Wallet),
		quit:    make(chan chan error),
	}

	if err := hub.readPairings(); err != nil {
		return nil, err
	}

	hub.refreshWallets()
	return hub, nil
}

// Wallets implements accounts.Backend, returning all the currently tracked USB
// devices that appear to be hardware wallets.
func (hub *Hub) Wallets() []accounts.Wallet {
	// Make sure the list of wallets is up to date
	hub.stateLock.Lock()
	defer hub.stateLock.Unlock()

	hub.refreshWallets()

	cpy := make([]accounts.Wallet, 0, len(hub.wallets))
	for _, wallet := range hub.wallets {
		if wallet != nil {
			cpy = append(cpy, wallet)
		}
	}
	return cpy
}

// refreshWallets scans the USB devices attached to the machine and updates the
// list of wallets based on the found devices.
func (hub *Hub) refreshWallets() {
	elapsed := time.Since(hub.refreshed)
	if elapsed < refreshThrottling {
		return
	}

	readers, err := hub.context.ListReaders()
	if err != nil {
		log.Error("Error listing readers", "err", err)
	}

	events := []accounts.WalletEvent{}
	seen := make(map[string]struct{})
	for _, reader := range readers {
		if wallet, ok := hub.wallets[reader]; ok {
			// We already know about this card; check it's still present
			if err := wallet.ping(); err != nil {
				log.Debug("Got error pinging wallet", "reader", reader, "err", err)
			} else {
				seen[reader] = struct{}{}
			}
			continue
		}
		seen[reader] = struct{}{}

		card, err := hub.context.Connect(reader, scard.ShareShared, scard.ProtocolAny)
		if err != nil {
			log.Debug("Error opening card", "reader", reader, "err", err)
			continue
		}

		wallet := NewWallet(hub, card)
		err = wallet.connect()
		if err != nil {
			log.Debug("Error connecting to wallet", "reader", reader, "err", err)
			card.Disconnect(scard.LeaveCard)
			continue
		}

		hub.wallets[reader] = wallet
		events = append(events, accounts.WalletEvent{Wallet: wallet, Kind: accounts.WalletArrived})
		log.Info("Found new smartcard wallet", "reader", reader, "publicKey", hexutil.Encode(wallet.PublicKey[:4]))
	}

	// Remove any wallets we no longer see
	for k, wallet := range hub.wallets {
		if _, ok := seen[k]; !ok {
			log.Info("Wallet disconnected", "pubkey", hexutil.Encode(wallet.PublicKey[:4]), "reader", k)
			wallet.Close()
			events = append(events, accounts.WalletEvent{Wallet: wallet, Kind: accounts.WalletDropped})
			delete(hub.wallets, k)
		}
	}

	for _, event := range events {
		hub.updateFeed.Send(event)
	}
	hub.refreshed = time.Now()
}

// Subscribe implements accounts.Backend, creating an async subscription to
// receive notifications on the addition or removal of wallets.
func (hub *Hub) Subscribe(sink chan<- accounts.WalletEvent) event.Subscription {
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

// updater is responsible for maintaining an up-to-date list of wallets managed
// by the hub, and for firing wallet addition/removal events.
func (hub *Hub) updater() {
	for {
		time.Sleep(refreshCycle)

		// Run the wallet refresher
		hub.stateLock.Lock()
		hub.refreshWallets()

		// If all our subscribers left, stop the updater
		if hub.updateScope.Count() == 0 {
			hub.updating = false
			hub.stateLock.Unlock()
			return
		}
		hub.stateLock.Unlock()
	}
}
