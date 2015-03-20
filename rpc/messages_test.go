package rpc

import (
	"testing"
)

func TestInsufficientParamsError(t *testing.T) {
	err := NewInsufficientParamsError(0, 1)
	expected := "insufficient params, want 1 have 0"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestNotImplementedError(t *testing.T) {
	err := NewNotImplementedError("foo")
	expected := "foo method not implemented"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestDecodeParamError(t *testing.T) {
	err := NewDecodeParamError("foo")
	expected := "could not decode, foo"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("foo", "should be `bar`")
	expected := "foo not valid, should be `bar`"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}
