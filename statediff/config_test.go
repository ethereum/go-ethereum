package statediff_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/testhelpers"
)

func TestNewMode(t *testing.T) {
	mode, err := statediff.NewMode("csv")
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	if mode != statediff.CSV {
		t.Error()
	}

	_, err = statediff.NewMode("not a real mode")
	if err == nil {
		t.Error("Expected an error, and got nil.")
	}
}
