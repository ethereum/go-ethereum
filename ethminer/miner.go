package ethminer

import (
	"bytes"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
)

type Miner struct {
	pow       ethchain.PoW
	ethereum  ethchain.EthManager
	coinbase  []byte
	reactChan chan ethutil.React
	txs       []*ethchain.Transaction
	uncles    []*ethchain.Block
	block     *ethchain.Block
	powChan   chan []byte
	quitChan  chan ethutil.React
}

func NewDefaultMiner(coinbase []byte, ethereum ethchain.EthManager) Miner {
	reactChan := make(chan ethutil.React, 1) // This is the channel that receives 'updates' when ever a new transaction or block comes in
	powChan := make(chan []byte, 1)          // This is the channel that receives valid sha hases for a given block
	quitChan := make(chan ethutil.React, 1)  // This is the channel that can exit the miner thread

	ethereum.Reactor().Subscribe("newBlock", reactChan)
	ethereum.Reactor().Subscribe("newTx:pre", reactChan)

	// We need the quit chan to be a Reactor event.
	// The POW search method is actually blocking and if we don't
	// listen to the reactor events inside of the pow itself
	// The miner overseer will never get the reactor events themselves
	// Only after the miner will find the sha
	ethereum.Reactor().Subscribe("newBlock", quitChan)
	ethereum.Reactor().Subscribe("newTx:pre", quitChan)

	miner := Miner{
		pow:       &ethchain.EasyPow{},
		ethereum:  ethereum,
		coinbase:  coinbase,
		reactChan: reactChan,
		powChan:   powChan,
		quitChan:  quitChan,
	}

	// Insert initial TXs in our little miner 'pool'
	miner.txs = ethereum.TxPool().Flush()
	miner.block = ethereum.BlockChain().NewBlock(miner.coinbase, miner.txs)

	return miner
}
func (miner *Miner) Start() {
	// Prepare inital block
	//miner.ethereum.StateManager().Prepare(miner.block.State(), miner.block.State())
	go miner.listener()
}
func (miner *Miner) listener() {
	for {
		select {
		case chanMessage := <-miner.reactChan:
			if block, ok := chanMessage.Resource.(*ethchain.Block); ok {
				ethutil.Config.Log.Infoln("[MINER] Got new block via Reactor")
				if bytes.Compare(miner.ethereum.BlockChain().CurrentBlock.Hash(), block.Hash()) == 0 {
					// TODO: Perhaps continue mining to get some uncle rewards
					ethutil.Config.Log.Infoln("[MINER] New top block found resetting state")

					// Filter out which Transactions we have that were not in this block
					var newtxs []*ethchain.Transaction
					for _, tx := range miner.txs {
						found := false
						for _, othertx := range block.Transactions() {
							if bytes.Compare(tx.Hash(), othertx.Hash()) == 0 {
								found = true
							}
						}
						if found == false {
							newtxs = append(newtxs, tx)
						}
					}
					miner.txs = newtxs

					// Setup a fresh state to mine on
					miner.block = miner.ethereum.BlockChain().NewBlock(miner.coinbase, miner.txs)

				} else {
					if bytes.Compare(block.PrevHash, miner.ethereum.BlockChain().CurrentBlock.PrevHash) == 0 {
						ethutil.Config.Log.Infoln("[MINER] Adding uncle block")
						miner.uncles = append(miner.uncles, block)
						//miner.ethereum.StateManager().Prepare(miner.block.State(), miner.block.State())
					}
				}
			}

			if tx, ok := chanMessage.Resource.(*ethchain.Transaction); ok {
				//log.Infoln("[MINER] Got new transaction from Reactor", tx)
				found := false
				for _, ctx := range miner.txs {
					if found = bytes.Compare(ctx.Hash(), tx.Hash()) == 0; found {
						break
					}

				}
				if found == false {
					miner.block.Undo()
					//log.Infoln("[MINER] We did not know about this transaction, adding")
					miner.txs = append(miner.txs, tx)
					miner.block = miner.ethereum.BlockChain().NewBlock(miner.coinbase, miner.txs)
					miner.block.SetTransactions(miner.txs)
				} else {
					//log.Infoln("[MINER] We already had this transaction, ignoring")
				}
			}
		default:
			ethutil.Config.Log.Infoln("[MINER] Mining on block. Includes", len(miner.txs), "transactions")

			// Apply uncles
			if len(miner.uncles) > 0 {
				miner.block.SetUncles(miner.uncles)
			}

			// Apply all transactions to the block
			miner.ethereum.StateManager().ApplyTransactions(miner.block.State(), miner.block, miner.block.Transactions())
			miner.ethereum.StateManager().AccumelateRewards(miner.block.State(), miner.block)

			// Search the nonce
			miner.block.Nonce = miner.pow.Search(miner.block, miner.quitChan)
			if miner.block.Nonce != nil {
				err := miner.ethereum.StateManager().Process(miner.block, true)
				if err != nil {
					ethutil.Config.Log.Infoln(err)
					miner.txs = []*ethchain.Transaction{} // Move this somewhere neat
					miner.block = miner.ethereum.BlockChain().NewBlock(miner.coinbase, miner.txs)
				} else {
					miner.ethereum.Broadcast(ethwire.MsgBlockTy, []interface{}{miner.block.Value().Val})
					ethutil.Config.Log.Infof("[MINER] 🔨  Mined block %x\n", miner.block.Hash())

					miner.txs = []*ethchain.Transaction{} // Move this somewhere neat
					miner.block = miner.ethereum.BlockChain().NewBlock(miner.coinbase, miner.txs)
				}
			}
		}
	}
}
