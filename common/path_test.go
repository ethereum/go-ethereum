package common

import (
	"os"
	// "testing"

	checker "gopkg.in/check.v1"
)

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
