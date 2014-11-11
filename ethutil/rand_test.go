package ethutil

import (
	checker "gopkg.in/check.v1"
)

type RandomSuite struct{}

var _ = checker.Suite(&RandomSuite{})

func (s *RandomSuite) TestRandomUint64(c *checker.C) {
	res1, _ := RandomUint64()
	res2, _ := RandomUint64()
	c.Assert(res1, checker.NotNil)
	c.Assert(res2, checker.NotNil)
	c.Assert(res1, checker.Not(checker.Equals), res2)
}
