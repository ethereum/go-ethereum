package chain

import (
	"container/list"
	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/wire"
	"testing"
)

func newChainManager() *ChainManager {
	bc := &ChainManager{}
	bc.genesisBlock = types.NewBlockFromBytes(ethutil.Encode(Genesis))
	bc.Reset()
	return bc
}

type fakeEth struct{}

func (e *fakeEth) BlockManager() *BlockManager                        { return nil }
func (e *fakeEth) ChainManager() *ChainManager                        { return newChainManager() }
func (e *fakeEth) TxPool() *TxPool                                    { return &TxPool{} }
func (e *fakeEth) Broadcast(msgType wire.MsgType, data []interface{}) {}
func (e *fakeEth) PeerCount() int                                     { return 0 }
func (e *fakeEth) IsMining() bool                                     { return false }
func (e *fakeEth) IsListening() bool                                  { return false }
func (e *fakeEth) Peers() *list.List                                  { return nil }
func (e *fakeEth) KeyManager() *crypto.KeyManager                     { return nil }
func (e *fakeEth) ClientIdentity() wire.ClientIdentity                { return nil }
func (e *fakeEth) Db() ethutil.Database                               { return nil }
func (e *fakeEth) EventMux() *event.TypeMux                           { return nil }

func TestValidateTransaction(t *testing.T) {
	// this is a well formed transaction with a made up signature
	badSig := "f87180881bc16d674ec80000881bc16d674ec8000094bbbd0256041f7aed3ce278c56ee61492de96d0018401312d008061a06162636465666768696a6b6c6d6e6f707172737475767778797a616263646566a06162636465666768696a6b6c6d6e6f707172737475767778797a616263646566"

	pool := NewTxPool(new(fakeEth))

	tx := types.NewTransactionFromBytes(ethutil.Hex2Bytes(badSig))
	err := pool.ValidateTransaction(tx)
	if err == nil {
		t.Error("Expected an error")
	}
}
