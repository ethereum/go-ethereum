package ethchain

import (
	"container/list"
	"fmt"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethreact"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
)

// Implement our EthTest Manager
type TestManager struct {
	stateManager *StateManager
	reactor      *ethreact.ReactorEngine

	txPool     *TxPool
	blockChain *BlockChain
	Blocks     []*Block
}

func (s *TestManager) IsListening() bool {
	return false
}

func (s *TestManager) IsMining() bool {
	return false
}

func (s *TestManager) PeerCount() int {
	return 0
}

func (s *TestManager) Peers() *list.List {
	return list.New()
}

func (s *TestManager) BlockChain() *BlockChain {
	return s.blockChain
}

func (tm *TestManager) TxPool() *TxPool {
	return tm.txPool
}

func (tm *TestManager) StateManager() *StateManager {
	return tm.stateManager
}

func (tm *TestManager) Reactor() *ethreact.ReactorEngine {
	return tm.reactor
}
func (tm *TestManager) Broadcast(msgType ethwire.MsgType, data []interface{}) {
	fmt.Println("Broadcast not implemented")
}

func (tm *TestManager) ClientIdentity() ethwire.ClientIdentity {
	return nil
}
func (tm *TestManager) KeyManager() *ethcrypto.KeyManager {
	return nil
}

func (tm *TestManager) Db() ethutil.Database { return nil }
func NewTestManager() *TestManager {
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "ETH")

	db, err := ethdb.NewMemDatabase()
	if err != nil {
		fmt.Println("Could not create mem-db, failing")
		return nil
	}
	ethutil.Config.Db = db

	testManager := &TestManager{}
	testManager.reactor = ethreact.New()

	testManager.txPool = NewTxPool(testManager)
	testManager.blockChain = NewBlockChain(testManager)
	testManager.stateManager = NewStateManager(testManager)

	// Start the tx pool
	testManager.txPool.Start()

	return testManager
}
