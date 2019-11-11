// Copyright 2019 The go-ethereum Authors
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

package payment

import (
	"io"
	"math/big"
)

// Payment is the way the light client pays to the les server. Payment can be
// implemented in many different ways, such as off-chain payment, on-chain payment.
// All available payments must implement the following functions.
type Payment interface {
	// Pay initiates a payment to the designated payee with specified
	// payemnt amount.
	Pay(amount *big.Int) error

	// Receive receives a payment from the payer and returns any error
	// for payment processing and proving.
	Receive(msg io.Reader) error

	// Amend amends the local payment db based on the received message.
	Amend(msg io.Reader) error

	// Close exits the payment and opens the reqeust to withdraw all funds.
	Close() error
}
