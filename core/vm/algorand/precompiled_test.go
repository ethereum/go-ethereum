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
	"fmt"
	"testing"

	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// MockClient is a mock client for testing.
type MockClient struct {
	// store is mapping from Algorand addresses to Algorand accounts.
	store map[string]*models.Account
}

// NewMockClient creates a new mock client.
func NewMockClient() *MockClient {
	store := make(map[string]*models.Account)
	store["737777777777777777777777777777777777777777777777777UFEJ2CI"] = &models.Account{
		Address: "737777777777777777777777777777777777777777777777777UFEJ2CI",
		Amount:  10000000,
	}
	return &MockClient{
		store: store,
	}
}

// GetAccount returns the account information.
func (c *MockClient) GetAccount(address string) (*models.Account, error) {
	if _, ok := c.store[address]; !ok {
		return nil, fmt.Errorf("account not found")
	}
	return c.store[address], nil
}

// CheckStatus of MockClient is always successful.
func (c *MockClient) CheckStatus() error {
	return nil
}

// TestRun tests the Run function on an Algorand object with a MockClient.
func TestRun(t *testing.T) {
	algorand := &Algorand{
		algodClient: NewMockClient(),
	}
	rawInput, err := common.ParseHexOrString("0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000006416d6f756e740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003a3733373737373737373737373737373737373737373737373737373737373737373737373737373737373737373737373737375546454a324349000000000000")
	require.NoError(t, err)

	rawOutput, err := algorand.Run(rawInput)
	require.NoError(t, err)
	v := new(uint64)
	err = unpack(rawOutput, v)
	require.NoError(t, err)
	require.Equal(t, uint64(10000000), *v)
}
