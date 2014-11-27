package types

/*
import (
	"testing"

	"github.com/ethereum/go-ethereum/state"
)

func TestBloom9(t *testing.T) {
	testCase := []byte("testtest")
	bin := LogsBloom([]state.Log{
		{testCase, [][]byte{[]byte("hellohello")}, nil},
	}).Bytes()
	res := BloomLookup(bin, testCase)

	if !res {
		t.Errorf("Bloom lookup failed")
	}
}


func TestAddress(t *testing.T) {
	block := &Block{}
	block.Coinbase = ethutil.Hex2Bytes("22341ae42d6dd7384bc8584e50419ea3ac75b83f")
	fmt.Printf("%x\n", crypto.Sha3(block.Coinbase))

	bin := CreateBloom(block)
	fmt.Printf("bin = %x\n", ethutil.LeftPadBytes(bin, 64))
}
*/
