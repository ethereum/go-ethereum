package test

import (
	"fmt"
	"testing"
	"time"
)

func CheckInt(name string, got int, expected int, t *testing.T) (err error) {
	if got != expected {
		t.Errorf("status for %v incorrect. expected %v, got %v", name, expected, got)
		err = fmt.Errorf("")
	}
	return
}

func CheckDuration(name string, got time.Duration, expected time.Duration, t *testing.T) (err error) {
	if got != expected {
		t.Errorf("status for %v incorrect. expected %v, got %v", name, expected, got)
		err = fmt.Errorf("")
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
