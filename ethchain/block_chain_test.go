package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"testing"
)

// Implement our EthTest Manager
type TestManager struct {
	stateManager *StateManager
	reactor      *ethutil.ReactorEngine

	txPool     *TxPool
	blockChain *BlockChain
	Blocks     []*Block
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

func (tm *TestManager) Reactor() *ethutil.ReactorEngine {
	return tm.reactor
}
func (tm *TestManager) Broadcast(msgType ethwire.MsgType, data []interface{}) {
	fmt.Println("Broadcast not implemented")
}

func NewTestManager() *TestManager {
	ethutil.ReadConfig(".ethtest")

	db, err := ethdb.NewMemDatabase()
	if err != nil {
		fmt.Println("Could not create mem-db, failing")
		return nil
	}
	ethutil.Config.Db = db

	testManager := &TestManager{}
	testManager.reactor = ethutil.NewReactorEngine()

	testManager.txPool = NewTxPool(testManager)
	testManager.blockChain = NewBlockChain(testManager)
	testManager.stateManager = NewStateManager(testManager)

	// Start the tx pool
	testManager.txPool.Start()

	return testManager
}
func (tm *TestManager) AddFakeBlock(blk []byte) error {
	block := NewBlockFromBytes(blk)
	tm.Blocks = append(tm.Blocks, block)
	tm.StateManager().PrepareDefault(block)
	err := tm.StateManager().ProcessBlock(block, false)
	return err
}
func (tm *TestManager) CreateChain1() error {
	err := tm.AddFakeBlock([]byte{248, 246, 248, 242, 160, 58, 253, 98, 206, 198, 181, 152, 223, 201, 116, 197, 154, 111, 104, 54, 113, 249, 184, 246, 15, 226, 142, 187, 47, 138, 60, 201, 66, 226, 237, 29, 7, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 184, 65, 4, 103, 109, 19, 120, 219, 91, 248, 48, 204, 17, 28, 7, 146, 72, 203, 15, 207, 251, 31, 216, 138, 26, 59, 34, 238, 40, 114, 233, 1, 13, 207, 90, 71, 136, 124, 86, 196, 127, 10, 176, 193, 154, 165, 76, 155, 154, 59, 45, 34, 96, 183, 212, 99, 41, 27, 40, 119, 171, 231, 160, 114, 56, 218, 173, 160, 80, 218, 177, 253, 147, 35, 101, 59, 37, 87, 97, 193, 119, 21, 132, 111, 93, 53, 152, 203, 38, 134, 25, 104, 138, 236, 92, 27, 176, 89, 229, 176, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 131, 63, 240, 0, 132, 83, 48, 32, 251, 128, 160, 4, 10, 11, 225, 132, 86, 146, 227, 229, 137, 164, 245, 16, 139, 219, 12, 251, 178, 154, 168, 210, 18, 84, 40, 250, 41, 124, 92, 169, 242, 246, 180, 192, 192})
	err = tm.AddFakeBlock([]byte{248, 246, 248, 242, 160, 222, 229, 152, 228, 200, 163, 244, 144, 120, 18, 203, 253, 195, 185, 105, 131, 163, 226, 116, 40, 140, 68, 249, 198, 221, 152, 121, 0, 124, 11, 180, 125, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 184, 65, 4, 103, 109, 19, 120, 219, 91, 248, 48, 204, 17, 28, 7, 146, 72, 203, 15, 207, 251, 31, 216, 138, 26, 59, 34, 238, 40, 114, 233, 1, 13, 207, 90, 71, 136, 124, 86, 196, 127, 10, 176, 193, 154, 165, 76, 155, 154, 59, 45, 34, 96, 183, 212, 99, 41, 27, 40, 119, 171, 231, 160, 114, 56, 218, 173, 160, 80, 218, 177, 253, 147, 35, 101, 59, 37, 87, 97, 193, 119, 21, 132, 111, 93, 53, 152, 203, 38, 134, 25, 104, 138, 236, 92, 27, 176, 89, 229, 176, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 131, 63, 224, 4, 132, 83, 48, 36, 250, 128, 160, 79, 58, 51, 246, 238, 249, 210, 253, 136, 83, 71, 134, 49, 114, 190, 189, 242, 78, 100, 238, 101, 84, 204, 176, 198, 25, 139, 151, 60, 84, 51, 126, 192, 192})
	err = tm.AddFakeBlock([]byte{248, 246, 248, 242, 160, 68, 52, 33, 210, 160, 189, 217, 255, 78, 37, 196, 217, 94, 247, 166, 169, 224, 199, 102, 110, 85, 213, 45, 13, 173, 106, 4, 103, 151, 195, 38, 86, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 184, 65, 4, 103, 109, 19, 120, 219, 91, 248, 48, 204, 17, 28, 7, 146, 72, 203, 15, 207, 251, 31, 216, 138, 26, 59, 34, 238, 40, 114, 233, 1, 13, 207, 90, 71, 136, 124, 86, 196, 127, 10, 176, 193, 154, 165, 76, 155, 154, 59, 45, 34, 96, 183, 212, 99, 41, 27, 40, 119, 171, 231, 160, 114, 56, 218, 173, 160, 80, 218, 177, 253, 147, 35, 101, 59, 37, 87, 97, 193, 119, 21, 132, 111, 93, 53, 152, 203, 38, 134, 25, 104, 138, 236, 92, 27, 176, 89, 229, 176, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 131, 63, 208, 12, 132, 83, 48, 38, 206, 128, 160, 65, 147, 32, 128, 177, 198, 131, 57, 57, 68, 135, 65, 198, 178, 138, 43, 25, 135, 92, 174, 208, 119, 103, 225, 26, 207, 243, 31, 225, 29, 173, 119, 192, 192})
	return err
}
func (tm *TestManager) CreateChain2() error {
	err := tm.AddFakeBlock([]byte{248, 246, 248, 242, 160, 58, 253, 98, 206, 198, 181, 152, 223, 201, 116, 197, 154, 111, 104, 54, 113, 249, 184, 246, 15, 226, 142, 187, 47, 138, 60, 201, 66, 226, 237, 29, 7, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 184, 65, 4, 72, 201, 77, 81, 160, 103, 70, 18, 102, 204, 82, 192, 86, 157, 40, 30, 117, 218, 224, 202, 1, 36, 249, 88, 82, 210, 19, 156, 112, 31, 13, 117, 227, 0, 125, 221, 190, 165, 16, 193, 163, 161, 175, 33, 37, 184, 235, 62, 201, 93, 102, 185, 143, 54, 146, 114, 30, 253, 178, 245, 87, 38, 191, 214, 160, 80, 218, 177, 253, 147, 35, 101, 59, 37, 87, 97, 193, 119, 21, 132, 111, 93, 53, 152, 203, 38, 134, 25, 104, 138, 236, 92, 27, 176, 89, 229, 176, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 131, 63, 240, 0, 132, 83, 48, 40, 35, 128, 160, 162, 214, 119, 207, 212, 186, 64, 47, 14, 186, 98, 118, 203, 79, 172, 205, 33, 206, 225, 177, 225, 194, 98, 188, 63, 219, 13, 151, 47, 32, 204, 27, 192, 192})
	err = tm.AddFakeBlock([]byte{248, 246, 248, 242, 160, 0, 210, 76, 6, 13, 18, 219, 190, 18, 250, 23, 178, 198, 117, 254, 85, 14, 74, 104, 116, 56, 144, 116, 172, 14, 3, 236, 99, 248, 228, 142, 91, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 184, 65, 4, 72, 201, 77, 81, 160, 103, 70, 18, 102, 204, 82, 192, 86, 157, 40, 30, 117, 218, 224, 202, 1, 36, 249, 88, 82, 210, 19, 156, 112, 31, 13, 117, 227, 0, 125, 221, 190, 165, 16, 193, 163, 161, 175, 33, 37, 184, 235, 62, 201, 93, 102, 185, 143, 54, 146, 114, 30, 253, 178, 245, 87, 38, 191, 214, 160, 80, 218, 177, 253, 147, 35, 101, 59, 37, 87, 97, 193, 119, 21, 132, 111, 93, 53, 152, 203, 38, 134, 25, 104, 138, 236, 92, 27, 176, 89, 229, 176, 160, 29, 204, 77, 232, 222, 199, 93, 122, 171, 133, 181, 103, 182, 204, 212, 26, 211, 18, 69, 27, 148, 138, 116, 19, 240, 161, 66, 253, 64, 212, 147, 71, 131, 63, 255, 252, 132, 83, 48, 40, 74, 128, 160, 185, 20, 138, 2, 210, 15, 71, 144, 89, 167, 94, 155, 148, 118, 170, 157, 122, 70, 70, 114, 50, 221, 231, 8, 132, 167, 115, 239, 44, 245, 41, 226, 192, 192})
	return err
}

func TestNegativeBlockChainReorg(t *testing.T) {
	// We are resetting the database between creation so we need to cache our information
	testManager2 := NewTestManager()
	testManager2.CreateChain2()
	tm2Blocks := testManager2.Blocks

	testManager := NewTestManager()
	testManager.CreateChain1()
	oldState := testManager.BlockChain().CurrentBlock.State()

	if testManager.BlockChain().FindCanonicalChain(tm2Blocks, testManager.BlockChain().GenesisBlock().Hash()) != true {
		t.Error("I expected TestManager to have the longest chain, but it was TestManager2 instead.")
	}
	if testManager.BlockChain().CurrentBlock.State() != oldState {
		t.Error("I expected the top state to be the same as it was as before the reorg")
	}

}

func TestPositiveBlockChainReorg(t *testing.T) {
	testManager := NewTestManager()
	testManager.CreateChain1()
	tm1Blocks := testManager.Blocks

	testManager2 := NewTestManager()
	testManager2.CreateChain2()
	oldState := testManager2.BlockChain().CurrentBlock.State()

	if testManager2.BlockChain().FindCanonicalChain(tm1Blocks, testManager.BlockChain().GenesisBlock().Hash()) == true {
		t.Error("I expected TestManager to have the longest chain, but it was TestManager2 instead.")
	}
	if testManager2.BlockChain().CurrentBlock.State() == oldState {
		t.Error("I expected the top state to have been modified but it was not")
	}
}
