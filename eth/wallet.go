package eth

/*
import (
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type Account struct {
	w *Wallet
}

func (self *Account) Transact(to *Account, value, gas, price *big.Int, data []byte) error {
	return self.w.transact(self, to, value, gas, price, data)
}

func (self *Account) Address() []byte {
	return nil
}

func (self *Account) PrivateKey() *ecdsa.PrivateKey {
	return nil
}

type Wallet struct{}

func NewWallet() *Wallet {
	return &Wallet{}
}

func (self *Wallet) GetAccount(i int) *Account {
}

func (self *Wallet) transact(from, to *Account, value, gas, price *big.Int, data []byte) error {
	if from.PrivateKey() == nil {
		return errors.New("accounts is not owned (no private key available)")
	}

	var createsContract bool
	if to == nil {
		createsContract = true
	}

	var msg *types.Transaction
	if contractCreation {
		msg = types.NewContractCreationTx(value, gas, price, data)
	} else {
		msg = types.NewTransactionMessage(to.Address(), value, gas, price, data)
	}

	state := self.chainManager.TransState()
	nonce := state.GetNonce(key.Address())

	msg.SetNonce(nonce)
	msg.SignECDSA(from.PriateKey())

	// Do some pre processing for our "pre" events  and hooks
	block := self.chainManager.NewBlock(from.Address())
	coinbase := state.GetOrNewStateObject(from.Address())
	coinbase.SetGasPool(block.GasLimit())
	self.blockManager.ApplyTransactions(coinbase, state, block, types.Transactions{tx}, true)

	err := self.obj.TxPool().Add(tx)
	if err != nil {
		return nil, err
	}
	state.SetNonce(key.Address(), nonce+1)

	if contractCreation {
		addr := core.AddressFromMessage(tx)
		pipelogger.Infof("Contract addr %x\n", addr)
	}

	return tx, nil
}
*/
