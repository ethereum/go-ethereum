package poseidon

import (
	"fmt"
	"testing"
)

func TestPoseidonCodeHash(t *testing.T) {
	// nil
	got := fmt.Sprintf("%s", CodeHash(nil))
	want := "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

	// single byte
	got = fmt.Sprintf("%s", CodeHash([]byte{0}))
	want = "0x0ee069e6aa796ef0e46cbd51d10468393d443a00f5affe72898d9ab62e335e16"

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
