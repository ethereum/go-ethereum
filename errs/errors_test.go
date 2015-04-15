package errs

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/logger"
)

func testErrors() *Errors {
	return &Errors{
		Package: "TEST",
		Errors: map[int]string{
			0: "zero",
			1: "one",
		},
		Level: func(i int) (l logger.LogLevel) {
			if i == 0 {
				l = logger.ErrorLevel
			} else {
				l = logger.WarnLevel
			}
			return
		},
	}
}

func TestErrorMessage(t *testing.T) {
	err := testErrors().New(0, "zero detail %v", "available")
	message := fmt.Sprintf("%v", err)
	exp := "[TEST] ERROR: zero: zero detail available"
	if message != exp {
		t.Errorf("error message incorrect. expected %v, got %v", exp, message)
	}
}

func TestErrorSeverity(t *testing.T) {
	err0 := testErrors().New(0, "zero detail")
	if !err0.Fatal() {
		t.Errorf("error should be fatal")
	}
	err1 := testErrors().New(1, "one detail")
	if err1.Fatal() {
		t.Errorf("error should not be fatal")
	}
}
