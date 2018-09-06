package mru

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func getTestUpdateLookup() *UpdateLookup {
	metadata := *getTestMetadata()
	rootAddr, _, _, _ := metadata.serializeAndHash()
	return &UpdateLookup{
		period:   79,
		version:  2010,
		rootAddr: rootAddr,
	}
}

func compareUpdateLookup(a, b *UpdateLookup) bool {
	return a.version == b.version &&
		a.period == b.period &&
		bytes.Equal(a.rootAddr, b.rootAddr)
}

func TestUpdateLookupUpdateAddr(t *testing.T) {
	ul := getTestUpdateLookup()
	updateAddr := ul.UpdateAddr()
	compareByteSliceToExpectedHex(t, "updateAddr", updateAddr, "0x8fbc8d4777ef6da790257eda80ab4321fabd08cbdbe67e4e3da6caca386d64e0")
}

func TestUpdateLookupSerializer(t *testing.T) {
	serializedUpdateLookup := make([]byte, updateLookupLength)
	ul := getTestUpdateLookup()
	if err := ul.binaryPut(serializedUpdateLookup); err != nil {
		t.Fatal(err)
	}
	compareByteSliceToExpectedHex(t, "serializedUpdateLookup", serializedUpdateLookup, "0x4f000000da070000fb0ed7efa696bdb0b54cd75554cc3117ffc891454317df7dd6fefad978e2f2fb")

	// set receiving slice to the wrong size
	serializedUpdateLookup = make([]byte, updateLookupLength+7)
	if err := ul.binaryPut(serializedUpdateLookup); err == nil {
		t.Fatalf("Expected UpdateLookup.binaryPut to fail when receiving slice has a length != %d", updateLookupLength)
	}

	// set rootAddr to an invalid length
	ul.rootAddr = []byte{1, 2, 3, 4}
	serializedUpdateLookup = make([]byte, updateLookupLength)
	if err := ul.binaryPut(serializedUpdateLookup); err == nil {
		t.Fatal("Expected UpdateLookup.binaryPut to fail when rootAddr is not of the correct size")
	}
}

func TestUpdateLookupDeserializer(t *testing.T) {
	serializedUpdateLookup, _ := hexutil.Decode("0x4f000000da070000fb0ed7efa696bdb0b54cd75554cc3117ffc891454317df7dd6fefad978e2f2fb")
	var recoveredUpdateLookup UpdateLookup
	if err := recoveredUpdateLookup.binaryGet(serializedUpdateLookup); err != nil {
		t.Fatal(err)
	}
	originalUpdateLookup := *getTestUpdateLookup()
	if !compareUpdateLookup(&originalUpdateLookup, &recoveredUpdateLookup) {
		t.Fatalf("Expected recovered UpdateLookup to match")
	}

	// set source slice to the wrong size
	serializedUpdateLookup = make([]byte, updateLookupLength+4)
	if err := recoveredUpdateLookup.binaryGet(serializedUpdateLookup); err == nil {
		t.Fatalf("Expected UpdateLookup.binaryGet to fail when source slice has a length != %d", updateLookupLength)
	}
}

func TestUpdateLookupSerializeDeserialize(t *testing.T) {
	serializedUpdateLookup := make([]byte, updateLookupLength)
	originalUpdateLookup := getTestUpdateLookup()
	if err := originalUpdateLookup.binaryPut(serializedUpdateLookup); err != nil {
		t.Fatal(err)
	}
	var recoveredUpdateLookup UpdateLookup
	if err := recoveredUpdateLookup.binaryGet(serializedUpdateLookup); err != nil {
		t.Fatal(err)
	}
	if !compareUpdateLookup(originalUpdateLookup, &recoveredUpdateLookup) {
		t.Fatalf("Expected recovered UpdateLookup to match")
	}
}
