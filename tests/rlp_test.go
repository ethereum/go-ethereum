package tests

import (
	"path/filepath"
	"testing"
)

func TestRLP(t *testing.T) {
	err := RunRLPTest(filepath.Join(rlpTestDir, "rlptest.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRLP_invalid(t *testing.T) {
	err := RunRLPTest(filepath.Join(rlpTestDir, "invalidRLPTest.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
}
