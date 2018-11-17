package feed

import (
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
)

func getTestID() *ID {
	return &ID{
		Feed:  *getTestFeed(),
		Epoch: lookup.GetFirstEpoch(1000),
	}
}

func TestIDAddr(t *testing.T) {
	id := getTestID()
	updateAddr := id.Addr()
	compareByteSliceToExpectedHex(t, "updateAddr", updateAddr, "0x8b24583ec293e085f4c78aaee66d1bc5abfb8b4233304d14a349afa57af2a783")
}

func TestIDSerializer(t *testing.T) {
	testBinarySerializerRecovery(t, getTestID(), "0x776f726c64206e657773207265706f72742c20657665727920686f7572000000876a8936a7cd0b79ef0735ad0896c1afe278781ce803000000000019")
}

func TestIDLengthCheck(t *testing.T) {
	testBinarySerializerLengthCheck(t, getTestID())
}
