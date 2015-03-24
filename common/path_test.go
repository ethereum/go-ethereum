package common

import (
	"os"
	"testing"

	checker "gopkg.in/check.v1"
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

type CommonSuite struct{}

var _ = checker.Suite(&CommonSuite{})

func (s *CommonSuite) TestOS(c *checker.C) {
	expwin := (os.PathSeparator == '\\' && os.PathListSeparator == ';')
	res := IsWindows()

	if !expwin {
		c.Assert(res, checker.Equals, expwin, checker.Commentf("IsWindows is", res, "but path is", os.PathSeparator))
	} else {
		c.Assert(res, checker.Not(checker.Equals), expwin, checker.Commentf("IsWindows is", res, "but path is", os.PathSeparator))
	}
}

func (s *CommonSuite) TestWindonziePath(c *checker.C) {
	iswindowspath := os.PathSeparator == '\\'
	path := "/opt/eth/test/file.ext"
	res := WindonizePath(path)
	ressep := string(res[0])

	if !iswindowspath {
		c.Assert(ressep, checker.Equals, "/")
	} else {
		c.Assert(ressep, checker.Not(checker.Equals), "/")
	}
}
