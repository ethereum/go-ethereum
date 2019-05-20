// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

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
