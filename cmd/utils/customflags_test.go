package utils

import (
	"os"
	"os/user"
	"testing"
)

func TestPathExpansion(t *testing.T) {

	user, _ := user.Current()

	tests := map[string]string{
		"/home/someuser/tmp": "/home/someuser/tmp",
		"~/tmp":              user.HomeDir + "/tmp",
		"$DDDXXX/a/b":        "/tmp/a/b",
		"/a/b/":              "/a/b",
	}

	os.Setenv("DDDXXX", "/tmp")

	for test, expected := range tests {
		got := expandPath(test)
		if got != expected {
			t.Errorf("test %s, got %s, expected %s\n", test, got, expected)
		}
	}
}
