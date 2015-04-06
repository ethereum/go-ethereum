package test

import (
	"fmt"
	"testing"
	"time"
)

// miscellaneous test helpers

func CheckInt(name string, got int, expected int, t *testing.T) (err error) {
	if got != expected {
		err = fmt.Errorf("status for %v incorrect. expected %v, got %v", name, expected, got)
		if t != nil {
			t.Error(err)
		}
	}
	return
}

func CheckDuration(name string, got time.Duration, expected time.Duration, t *testing.T) (err error) {
	if got != expected {
		err = fmt.Errorf("status for %v incorrect. expected %v, got %v", name, expected, got)
		if t != nil {
			t.Error(err)
		}
	}
	return
}

func ArrayEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
