package presets

import "testing"

func TestPresetsValidate(t *testing.T) {
	if err := Mainnet.Validate(); err != nil {
		t.Fatal("mainnet is invalid:", err)
	}
	if err := Sepolia.Validate(); err != nil {
		t.Fatal("sepolia is invalid:", err)
	}
}
