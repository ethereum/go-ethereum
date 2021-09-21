package privacy

import (
	"fmt"
	"testing"
	)

func TestSign(t *testing.T) {
	/*for i := 14; i < 15; i++ {
	for j := 14; j < 15; j++ {
		for k := 0; k <= j; k++ {*/
	numRing := 5
	ringSize := 10
	s := 9
	fmt.Println("Generate random ring parameter ")
	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)

	fmt.Println("numRing  ", numRing)
	fmt.Println("ringSize  ", ringSize)
	fmt.Println("index of real one  ", s)

	fmt.Println("Ring  ", rings)
	fmt.Println("privkeys  ", privkeys)
	fmt.Println("m  ", m)

	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		t.Error("Failed to create Ring signature")
	}

	sig, err := ringSignature.Serialize()
	if err != nil {
		t.Error("Failed to Serialize input Ring signature")
	}

	deserializedSig, err := Deserialize(sig)
	if err != nil {
		t.Error("Failed to Deserialize Ring signature")
	}
	verified := Verify(deserializedSig, false)

	if !verified {
		t.Error("Failed to verify Ring signature")
	}

}
