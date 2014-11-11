package ethutil

import (
	checker "gopkg.in/check.v1"
)

type SizeSuite struct{}

var _ = checker.Suite(&SizeSuite{})

func (s *SizeSuite) TestStorageSizeString(c *checker.C) {
	data1 := 2381273
	data2 := 2192
	data3 := 12

	exp1 := "2.38 mB"
	exp2 := "2.19 kB"
	exp3 := "12.00 B"

	res1 := StorageSize(data1).String()
	res2 := StorageSize(data2).String()
	res3 := StorageSize(data3).String()

	if res1 != exp1 {
		t.Errorf("Expected %s got %s", exp1, res1)
	}

	if res2 != exp2 {
		t.Errorf("Expected %s got %s", exp2, res2)
	}

	if res3 != exp3 {
		t.Errorf("Expected %s got %s", exp3, res3)
	}
}
