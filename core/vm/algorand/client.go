// Copyright 2023 The go-ethereum Authors
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

package algorand

import (
	"context"
	"fmt"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
)

// Client is the interface for the Algorand client.
type Client interface {
	// GetAccount returns the account information.
	GetAccount(address string) (*models.Account, error)
	// CheckStatus checks the status of the client.
	CheckStatus() error
}

// AlgorandClient implements the Client interface.
type AlgorandClient struct {
	algodAddress string
	algodToken   string
	algodClient  *algod.Client
}

// NewClient creates a new Algorand client.
func NewClient(algodAddress, algodToken string) *AlgorandClient {
	algorandClient := &AlgorandClient{
		algodAddress: algodAddress,
		algodToken:   algodToken,
	}
	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err == nil {
		algorandClient.algodClient = algodClient
	}
	return algorandClient
}

func (c *AlgorandClient) CheckStatus() error {
	if c.algodAddress == "" {
		return fmt.Errorf("algodAddress is not set")
	}
	if c.algodToken == "" {
		return fmt.Errorf("algodToken is not set")
	}
	if c.algodClient == nil {
		return fmt.Errorf("algodClient is not initialized")
	}
	return nil
}

// GetAccount returns the account information.
func (c *AlgorandClient) GetAccount(address string) (*models.Account, error) {
	account, err := c.algodClient.AccountInformation(address).Do(context.Background())
	return &account, err
}
