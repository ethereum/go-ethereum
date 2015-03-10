package math

import (
	"fmt"
	"math/big"
	"testing"
)

type summer struct {
	numbers []*big.Int
}

func (s summer) Len() int { return len(s.numbers) }
func (s summer) Sum(i int) *big.Int {
	return s.numbers[i]
}

func TestSum(t *testing.T) {
	summer := summer{numbers: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}}
	sum := Sum(summer)
	if sum.Cmp(big.NewInt(6)) != 0 {
		t.Errorf("not 6", sum)
	}
}

func TestDist(t *testing.T) {
	var vectors = []Vector{
		Vector{big.NewInt(1000), big.NewInt(1234)},
		Vector{big.NewInt(500), big.NewInt(10023)},
		Vector{big.NewInt(1034), big.NewInt(1987)},
		Vector{big.NewInt(1034), big.NewInt(1987)},
		Vector{big.NewInt(8983), big.NewInt(1977)},
		Vector{big.NewInt(98382), big.NewInt(1887)},
		Vector{big.NewInt(12398), big.NewInt(1287)},
		Vector{big.NewInt(12398), big.NewInt(1487)},
		Vector{big.NewInt(12398), big.NewInt(1987)},
		Vector{big.NewInt(12398), big.NewInt(128)},
		Vector{big.NewInt(12398), big.NewInt(1987)},
		Vector{big.NewInt(1398), big.NewInt(187)},
		Vector{big.NewInt(12328), big.NewInt(1927)},
		Vector{big.NewInt(12398), big.NewInt(1987)},
		Vector{big.NewInt(22398), big.NewInt(1287)},
		Vector{big.NewInt(1370), big.NewInt(1981)},
		Vector{big.NewInt(12398), big.NewInt(1957)},
		Vector{big.NewInt(42198), big.NewInt(1987)},
	}

	VectorsBy(GasSort).Sort(vectors)
	fmt.Println(vectors)

	BP := big.NewInt(15)
	GL := big.NewInt(1000000)
	EP := big.NewInt(100)
	fmt.Println("BP", BP, "GL", GL, "EP", EP)
	GP := GasPrice(BP, GL, EP)
	fmt.Println("GP =", GP, "Wei per GU")

	S := len(vectors) / 4
	fmt.Println("L", len(vectors), "S", S)
	for i := 1; i <= S*4; i += S {
		fmt.Printf("T%d = %v\n", i, vectors[i])
	}

	g := VectorSum(GasSum).Sum(vectors)
	fmt.Printf("G = ∑g* (%v)\n", g)
}
