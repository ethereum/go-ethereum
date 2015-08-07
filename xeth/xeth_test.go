package xeth

import "testing"

func TestIsAddress(t *testing.T) {
	for _, invalid := range []string{
		"0x00",
		"0xNN",
		"0x00000000000000000000000000000000000000NN",
		"0xAAar000000000000000000000000000000000000",
	} {
		if isAddress(invalid) {
			t.Error("Expected", invalid, "to be invalid")
		}
	}

	for _, valid := range []string{
		"0x0000000000000000000000000000000000000000",
		"0xAABBbbCCccff9900000000000000000000000000",
		"AABBbbCCccff9900000000000000000000000000",
	} {
		if !isAddress(valid) {
			t.Error("Expected", valid, "to be valid")
		}
	}
}
