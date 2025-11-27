package usbwallet

import (
	"fmt"
	"testing"

	"github.com/karalabe/hid"
)

func TestWallets(t *testing.T) {
	devices := make([]hid.DeviceInfo, 0)
	for i, productID := range ledgerProductIDs {
		devices = append(devices, hid.DeviceInfo{
			ProductID: productID, UsagePage: deviceUsagePage, Interface: deviceInterface,
			Path: fmt.Sprintf("/dev/hidraw%d", i),
		})
	}
	usbEnumerate = func(vendorID, productID uint16) ([]hid.DeviceInfo, error) { return devices, nil }
	defer func() { usbEnumerate = hid.Enumerate }()

	hub, err := NewLedgerHub()
	if err != nil {
		t.Fatalf("Failed to create hub: %v", err)
	}
	wallets := hub.Wallets()
	if len(wallets) != len(devices) {
		t.Errorf("Expected %d wallets, got %d", len(devices), len(wallets))
	}
}
