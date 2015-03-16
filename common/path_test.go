package common

import (
	// "os"
	"testing"
)

func TestGoodFile(t *testing.T) {
	goodpath := "~/goethereumtest.pass"
	path := ExpandHomePath(goodpath)
	contentstring := "3.14159265358979323846"

	err := WriteFile(path, []byte(contentstring))
	if err != nil {
		t.Error("Could not write file")
	}

	if !FileExist(path) {
		t.Error("File not found at", path)
	}

	v, err := ReadAllFile(path)
	if err != nil {
		t.Error("Could not read file", path)
	}
	if v != contentstring {
		t.Error("Expected", contentstring, "Got", v)
	}

}

func TestBadFile(t *testing.T) {
	badpath := "/this/path/should/not/exist/goethereumtest.fail"
	path := ExpandHomePath(badpath)
	contentstring := "3.14159265358979323846"

	err := WriteFile(path, []byte(contentstring))
	if err == nil {
		t.Error("Wrote file, but should not be able to", path)
	}

	if FileExist(path) {
		t.Error("Found file, but should not be able to", path)
	}

	v, err := ReadAllFile(path)
	if err == nil {
		t.Error("Read file, but should not be able to", v)
	}

}
