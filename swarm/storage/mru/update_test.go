package mru

import (
	"bytes"
	"testing"
)

const serializedUpdateHex = "0x490034004f000000da070000fb0ed7efa696bdb0b54cd75554cc3117ffc891454317df7dd6fefad978e2f2fbf74a10ce8f26ffc8bfaa07c3031a34b2c61f517955e7deb1592daccf96c69cf000456c20717565206c6565206d7563686f207920616e6461206d7563686f2c207665206d7563686f20792073616265206d7563686f"
const serializedUpdateMultihashHex = "0x490022004f000000da070000fb0ed7efa696bdb0b54cd75554cc3117ffc891454317df7dd6fefad978e2f2fbf74a10ce8f26ffc8bfaa07c3031a34b2c61f517955e7deb1592daccf96c69cf0011b200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1c1e1f20"

func getTestResourceUpdate() *resourceUpdate {
	return &resourceUpdate{
		updateHeader: *getTestUpdateHeader(false),
		data:         []byte("El que lee mucho y anda mucho, ve mucho y sabe mucho"),
	}
}

func getTestResourceUpdateMultihash() *resourceUpdate {
	return &resourceUpdate{
		updateHeader: *getTestUpdateHeader(true),
		data:         []byte{0x1b, 0x20, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 28, 30, 31, 32},
	}
}

func compareResourceUpdate(a, b *resourceUpdate) bool {
	return compareUpdateHeader(&a.updateHeader, &b.updateHeader) &&
		bytes.Equal(a.data, b.data)
}

func TestResourceUpdateSerializer(t *testing.T) {
	var serializedUpdateLength = len(serializedUpdateHex)/2 - 1 // hack to calculate the byte length out of the hex representation
	update := getTestResourceUpdate()
	serializedUpdate := make([]byte, serializedUpdateLength)
	if err := update.binaryPut(serializedUpdate); err != nil {
		t.Fatal(err)
	}
	compareByteSliceToExpectedHex(t, "serializedUpdate", serializedUpdate, serializedUpdateHex)

	// Test fail if update does not contain data
	update.data = nil
	if err := update.binaryPut(serializedUpdate); err == nil {
		t.Fatal("Expected resourceUpdate.binaryPut to fail since update does not contain data")
	}

	// Test fail if update is too big
	update.data = make([]byte, 10000)
	if err := update.binaryPut(serializedUpdate); err == nil {
		t.Fatal("Expected resourceUpdate.binaryPut to fail since update is too big")
	}

	// Test fail if passed slice is not of the exact size required for this update
	update.data = make([]byte, 1)
	if err := update.binaryPut(serializedUpdate); err == nil {
		t.Fatal("Expected resourceUpdate.binaryPut to fail since passed slice is not of the appropriate size")
	}

	// Test serializing a multihash update
	var serializedUpdateMultihashLength = len(serializedUpdateMultihashHex)/2 - 1 // hack to calculate the byte length out of the hex representation
	update = getTestResourceUpdateMultihash()
	serializedUpdate = make([]byte, serializedUpdateMultihashLength)
	if err := update.binaryPut(serializedUpdate); err != nil {
		t.Fatal(err)
	}
	compareByteSliceToExpectedHex(t, "serializedUpdate", serializedUpdate, serializedUpdateMultihashHex)

	// mess with the multihash to test it fails with a wrong multihash error
	update.data[1] = 79
	if err := update.binaryPut(serializedUpdate); err == nil {
		t.Fatal("Expected resourceUpdate.binaryPut to fail since data contains an invalid multihash")
	}

}
