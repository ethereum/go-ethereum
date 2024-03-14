package pbeth

import (
	"encoding/hex"
	"math/big"
	"time"
)

var b0 = big.NewInt(0)

func (b *Block) PreviousID() string {
	return hex.EncodeToString(b.Header.ParentHash)
}

func (b *Block) Time() time.Time {
	return b.Header.Timestamp.AsTime()
}

func (m *BigInt) Native() *big.Int {
	if m == nil {
		return b0
	}

	z := new(big.Int)
	z.SetBytes(m.Bytes)
	return z
}
