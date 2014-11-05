package ethutil

import (
	"fmt"
	"testing"
)

func TestMisc(t *testing.T) {
	a := Big("10")
	b := Big("57896044618658097711785492504343953926634992332820282019728792003956564819968")
	c := []byte{1, 2, 3, 4}
	fmt.Println(b)
	z := BitTest(a, 1)
	fmt.Println(z)

	U256(a)
	S256(a)

	U256(b)
	S256(b)

	BigD(c)
}

func TestBigMax(t *testing.T) {
	a := Big("10")
	b := Big("5")

	max1 := BigMax(a, b)
	if max1 != a {
		t.Errorf("Expected %d got %d", a, max1)
	}

	max2 := BigMax(b, a)
	if max2 != a {
		t.Errorf("Expected %d got %d", a, max2)
	}
}

func TestBigMin(t *testing.T) {
	a := Big("10")
	b := Big("5")

	min1 := BigMin(a, b)
	if min1 != b {
		t.Errorf("Expected %d got %d", b, min1)
	}

	min2 := BigMin(b, a)
	if min2 != b {
		t.Errorf("Expected %d got %d", b, min2)
	}
}

func TestBigCopy(t *testing.T) {
	a := Big("10")
	b := BigCopy(a)
	c := Big("1000000000000")
	y := BigToBytes(b, 16)
	z := BigToBytes(c, 16)
	fmt.Println(y)
	fmt.Println(z)
}
