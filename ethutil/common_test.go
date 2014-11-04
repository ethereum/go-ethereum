package ethutil

import (
	"math/big"
	"os"
	"testing"
)

func TestOS(t *testing.T) {
	res := IsWindows()

	if res && (os.PathSeparator != '\\' || os.PathListSeparator != ';') {
		t.Error("IsWindows is", res, "but path is", os.PathSeparator)
	}

	if !res && (os.PathSeparator == '\\' && os.PathListSeparator == ';') {
		t.Error("IsWindows is", res, "but path is", os.PathSeparator)
	}
}

func TestWindonziePath(t *testing.T) {
	path := "/opt/eth/test/file.ext"
	res := WindonizePath(path)
	iswindowspath := os.PathSeparator == '\\'

	if !iswindowspath && string(res[0]) != "/" {
		t.Error("Got", res)
	}

	if iswindowspath && string(res[0]) == "/" {
		t.Error("Got", res)
	}
}

func TestCommon(t *testing.T) {
	ether := CurrencyToString(BigPow(10, 19))
	finney := CurrencyToString(BigPow(10, 16))
	szabo := CurrencyToString(BigPow(10, 13))
	shannon := CurrencyToString(BigPow(10, 10))
	babbage := CurrencyToString(BigPow(10, 7))
	ada := CurrencyToString(BigPow(10, 4))
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

	if shannon != "10 Shannon" {
		t.Error("Got", shannon)
	}

	if babbage != "10 Babbage" {
		t.Error("Got", babbage)
	}

	if ada != "10 Ada" {
		t.Error("Got", ada)
	}

	if wei != "10 Wei" {
		t.Error("Got", wei)
	}
}
