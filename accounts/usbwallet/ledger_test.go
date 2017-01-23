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

/*
func TestLedgerHub(t *testing.T) {
	glog.SetV(6)
	glog.SetToStderr(true)

	// Create a USB hub watching for Ledger devices
	hub, err := NewLedgerHub()
	if err != nil {
		t.Fatalf("Failed to create Ledger hub: %v", err)
	}
	defer hub.Close()

	// Wait for events :P
	time.Sleep(time.Minute)
}
*/
/*
func TestLedger(t *testing.T) {
	// Create a USB context to access devices through
	ctx, err := usb.NewContext()
	defer ctx.Close()
	ctx.Debug(6)

	// List all of the Ledger wallets
	wallets, err := findLedgerWallets(ctx)
	if err != nil {
		t.Fatalf("Failed to list Ledger wallets: %v", err)
	}
	// Retrieve the address from every one of them
	for _, wallet := range wallets {
		// Retrieve the version of the wallet app
		ver, err := wallet.Version()
		if err != nil {
			t.Fatalf("Failed to retrieve wallet version: %v", err)
		}
		fmt.Printf("Ledger version: %s\n", ver)

		// Retrieve the address of the wallet
		addr, err := wallet.Address()
		if err != nil {
			t.Fatalf("Failed to retrieve wallet address: %v", err)
		}
		fmt.Printf("Ledger address: %x\n", addr)

		// Try to sign a transaction with the wallet
		unsigned := types.NewTransaction(1, common.HexToAddress("0xbabababababababababababababababababababa"), common.Ether, big.NewInt(20000), common.Shannon, nil)
		signed, err := wallet.Sign(unsigned)
		if err != nil {
			t.Fatalf("Failed to sign transactions: %v", err)
		}
		signer, err := types.Sender(types.NewEIP155Signer(big.NewInt(1)), signed)
		if err != nil {
			t.Fatalf("Failed to recover signer: %v", err)
		}
		fmt.Printf("Ledger signature by: %x\n", signer)
	}
}*/
