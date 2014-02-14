package ethchain

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"testing"
)

func TestAddressRetrieval(t *testing.T) {
	// TODO
	// 88f9b82462f6c4bf4a0fb15e5c3971559a316e7f
	key, _ := hex.DecodeString("3ecb44df2159c26e0f995712d4f39b6f6e499b40749b1cf1246c37f9516cb6a4")

	tx := &Transaction{
		Nonce:     0,
		Recipient: ZeroHash160,
		Value:     big.NewInt(0),
		Data:      nil,
	}
	//fmt.Printf("rlp %x\n", tx.RlpEncode())
	//fmt.Printf("sha rlp %x\n", tx.Hash())

	tx.Sign(key)

	//fmt.Printf("hex tx key %x\n", tx.PublicKey())
	//fmt.Printf("seder %x\n", tx.Sender())
}

func TestAddressRetrieval2(t *testing.T) {
	// TODO
	// 88f9b82462f6c4bf4a0fb15e5c3971559a316e7f
	key, _ := hex.DecodeString("3ecb44df2159c26e0f995712d4f39b6f6e499b40749b1cf1246c37f9516cb6a4")
	addr, _ := hex.DecodeString("944400f4b88ac9589a0f17ed4671da26bddb668b")
	tx := &Transaction{
		Nonce:     0,
		Recipient: addr,
		Value:     big.NewInt(1000),
		Data:      nil,
	}
	tx.Sign(key)
	//data, _ := hex.DecodeString("f85d8094944400f4b88ac9589a0f17ed4671da26bddb668b8203e8c01ca0363b2a410de00bc89be40f468d16e70e543b72191fbd8a684a7c5bef51dc451fa02d8ecf40b68f9c64ed623f6ee24c9c878943b812e1e76bd73ccb2bfef65579e7")
	//tx := NewTransactionFromData(data)
	fmt.Println(tx.RlpValue())

	fmt.Printf("rlp %x\n", tx.RlpEncode())
	fmt.Printf("sha rlp %x\n", tx.Hash())

	//tx.Sign(key)

	fmt.Printf("hex tx key %x\n", tx.PublicKey())
	fmt.Printf("seder %x\n", tx.Sender())
}
