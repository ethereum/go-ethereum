// Copyright 2024 The go-ethereum Authors
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

package walletevent

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
)

type (
	Listener struct {
		logger    logger
		ethClient ethereum.ChainStateReader
	}

	logger interface {
		Warn(msg string, ctx ...interface{})
		Info(msg string, ctx ...interface{})
	}
)

// NewListener creates a new wallet event listener.
func NewListener(logger logger, ethClient ethereum.ChainStateReader) *Listener {
	return &Listener{
		logger:    logger,
		ethClient: ethClient,
	}
}

// Listen creates a new event listener for accounts.WalletEvent
func (l *Listener) Listen(wallets []accounts.Wallet, events chan accounts.WalletEvent) {
	// Open any wallets already attached
	for _, wallet := range wallets {
		if err := wallet.Open(""); err != nil {
			l.logger.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
		}
	}
	// Listen for wallet event till termination
	for event := range events {
		switch event.Kind {
		case accounts.WalletArrived:
			if err := event.Wallet.Open(""); err != nil {
				l.logger.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
			}
		case accounts.WalletOpened:
			status, _ := event.Wallet.Status()
			l.logger.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

			var derivationPaths []accounts.DerivationPath
			if event.Wallet.URL().Scheme == "ledger" {
				derivationPaths = append(derivationPaths, accounts.LegacyLedgerBaseDerivationPath)
			}
			derivationPaths = append(derivationPaths, accounts.DefaultBaseDerivationPath)

			event.Wallet.SelfDerive(derivationPaths, l.ethClient)

		case accounts.WalletDropped:
			l.logger.Info("Old wallet dropped", "url", event.Wallet.URL())
			event.Wallet.Close()
		}
	}
}
