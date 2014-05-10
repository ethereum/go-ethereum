package ethutil

import (
	"math/big"
	"testing"
)

func TestCommon(t *testing.T) {
	ether := CurrencyToString(BigPow(10, 19))
	finney := CurrencyToString(BigPow(10, 16))
	szabo := CurrencyToString(BigPow(10, 13))
	vito := CurrencyToString(BigPow(10, 10))
	turing := CurrencyToString(BigPow(10, 7))
	eins := CurrencyToString(BigPow(10, 4))
	wei := CurrencyToString(big.NewInt(10))

	if ether != "10 Ether" {
		t.Error("Got", ether)
	}

	if finney != "10 Finney" {
		t.Error("Got", finney)
	}

	if szabo != "10 Szabo" {
		t.Error("Got", szabo)
	}

	if vito != "10 Vita" {
		t.Error("Got", vito)
	}

	if turing != "10 Turing" {
		t.Error("Got", turing)
	}

	if eins != "10 Eins" {
		t.Error("Got", eins)
	}

	if wei != "10 Wei" {
		t.Error("Got", wei)
	}
}
