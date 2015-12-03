package api

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
!! THIS IS CURRENTLY A PLACEHOLEDER (HACK) UNTIL PRC v2
!! https://github.com/ethereum/go-ethereum/pull/1912 is merged
!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
*/

var (
	defaultGasPrice = big.NewInt(10000000000000) //150000000000
	defaultGas      = big.NewInt(90000)          //500000
	addrReg         = regexp.MustCompile(`^(0x)?[a-fA-F0-9]{40}$`)
)

type ethApi struct {
	eth           *eth.Ethereum
	gpo           *eth.GasPriceOracle
	transactionMu sync.RWMutex
	transactMu    sync.RWMutex
	state         *state.StateDB
}

func NewEthApi(ethereum *eth.Ethereum) *ethApi {
	return &ethApi{
		eth: ethereum,
		gpo: eth.NewGasPriceOracle(ethereum),
	}
}

// subscribes to new head block events and
// waits until blockchain height is greater n at any time
// given the current head, waits for the next chain event
// sets the state to the current head
// loop is async and quit by closing the channel
// used in tests and JS console debug module to control advancing private chain manually
// Note: this is not threadsafe, only called in JS single process and tests
func (self *ethApi) UpdateState() (wait chan *big.Int) {
	wait = make(chan *big.Int)
	self.state, _ = state.New(self.eth.BlockChain().GetBlockByNumber(0).Root(), self.eth.ChainDb())

	go func() {
		eventSub := self.eth.EventMux().Subscribe(core.ChainHeadEvent{})
		defer eventSub.Unsubscribe()

		var m, n *big.Int
		var ok bool

		eventCh := eventSub.Chan()
		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					// Event subscription closed, set the channel to nil to stop spinning
					eventCh = nil
					continue
				}
				// A real event arrived, process if new head block assignment
				if event, ok := event.Data.(core.ChainHeadEvent); ok {
					m = event.Block.Number()
					if n != nil && n.Cmp(m) < 0 {
						wait <- n
						n = nil
					}
					statedb, err := state.New(event.Block.Root(), self.eth.ChainDb())
					if err != nil {
						glog.V(logger.Error).Infoln("Could not create new state: %v", err)
						return
					}
					self.state = statedb
				}
			case n, ok = <-wait:
				if !ok {
					return
				}
			}
		}
	}()
	return
}

func (self *ethApi) AtStateNum(num int64) registrar.Backend {
	var st *state.StateDB
	var err error
	switch num {
	case -2:
		st = self.eth.Miner().PendingState().Copy()
	default:
		if block := self.getBlockByHeight(num); block != nil {
			st, err = state.New(block.Root(), self.eth.ChainDb())
			if err != nil {
				return nil
			}
		} else {
			st, err = state.New(self.eth.BlockChain().GetBlockByNumber(0).Root(), self.eth.ChainDb())
			if err != nil {
				return nil
			}
		}
	}
	return registrar.Backend(&ethApi{
		eth:   self.eth,
		state: st,
	})
}
func (self *ethApi) GetTxReceipt(txhash common.Hash) *types.Receipt {
	return core.GetReceipt(self.eth.ChainDb(), txhash)
}

func (self *ethApi) StorageAt(addr, storageAddr string) string {
	return self.state.GetState(common.HexToAddress(addr), common.HexToHash(storageAddr)).Hex()
}

func (self *ethApi) CodeAt(address string) string {
	return common.ToHex(self.state.GetCode(common.HexToAddress(address)))
}

func (self *ethApi) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, dataStr string) (string, string, error) {
	statedb := self.state.Copy()
	var from *state.StateObject
	if len(fromStr) == 0 {
		accounts, err := self.eth.AccountManager().Accounts()
		if err != nil || len(accounts) == 0 {
			from = statedb.GetOrNewStateObject(common.Address{})
		} else {
			from = statedb.GetOrNewStateObject(accounts[0].Address)
		}
	} else {
		from = statedb.GetOrNewStateObject(common.HexToAddress(fromStr))
	}

	from.SetBalance(common.MaxBig)

	msg := callmsg{
		from:     from,
		gas:      common.Big(gasStr),
		gasPrice: common.Big(gasPriceStr),
		value:    common.Big(valueStr),
		data:     common.FromHex(dataStr),
	}
	if len(toStr) > 0 {
		addr := common.HexToAddress(toStr)
		msg.to = &addr
	}

	if msg.gas.Cmp(big.NewInt(0)) == 0 {
		msg.gas = big.NewInt(50000000)
	}

	if msg.gasPrice.Cmp(big.NewInt(0)) == 0 {
		msg.gasPrice = self.DefaultGasPrice()
	}

	header := self.CurrentBlock().Header()
	vmenv := core.NewEnv(statedb, self.eth.BlockChain(), msg, header)
	gp := new(core.GasPool).AddGas(common.MaxBig)
	res, gas, err := core.ApplyMessage(vmenv, msg, gp)
	return common.ToHex(res), gas.String(), err
}

func (self *ethApi) Transact(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {

	if len(toStr) > 0 && toStr != "0x" && !isAddress(toStr) {
		return "", errors.New("Invalid address")
	}

	var (
		from             = common.HexToAddress(fromStr)
		to               = common.HexToAddress(toStr)
		value            = common.Big(valueStr)
		gas              *big.Int
		price            *big.Int
		data             []byte
		contractCreation bool
	)

	if len(gasStr) == 0 {
		gas = DefaultGas()
	} else {
		gas = common.Big(gasStr)
	}

	if len(gasPriceStr) == 0 {
		price = self.DefaultGasPrice()
	} else {
		price = common.Big(gasPriceStr)
	}

	data = common.FromHex(codeStr)
	if len(toStr) == 0 {
		contractCreation = true
	}

	self.transactMu.Lock()
	defer self.transactMu.Unlock()

	var nonce uint64
	if len(nonceStr) != 0 {
		nonce = common.Big(nonceStr).Uint64()
	} else {
		state := self.eth.TxPool().State()
		nonce = state.GetNonce(from)
	}
	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreation(nonce, value, gas, price, data)
	} else {
		tx = types.NewTransaction(nonce, to, value, gas, price, data)
	}

	signed, err := self.sign(tx, from, false)
	if err != nil {
		return "", err
	}
	if err = self.eth.TxPool().Add(signed); err != nil {
		return "", err
	}

	if contractCreation {
		addr := crypto.CreateAddress(from, nonce)
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signed.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signed.Hash().Hex(), tx.To().Hex())
	}

	return signed.Hash().Hex(), nil
}

func (self *ethApi) sign(tx *types.Transaction, from common.Address, didUnlock bool) (*types.Transaction, error) {
	hash := tx.SigHash()
	sig, err := self.doSign(from, hash, didUnlock)
	if err != nil {
		return tx, err
	}
	return tx.WithSignature(sig)
}

func (self *ethApi) doSign(from common.Address, hash common.Hash, didUnlock bool) ([]byte, error) {
	sig, err := self.eth.AccountManager().Sign(accounts.Account{Address: from}, hash.Bytes())
	if err == accounts.ErrLocked {
		if didUnlock {
			return nil, fmt.Errorf("signer account still locked after successful unlock")
		}
		// retry signing, the account should now be unlocked.
		return self.doSign(from, hash, true)
	} else if err != nil {
		return nil, err
	}
	return sig, nil
}

func DefaultGas() *big.Int { return new(big.Int).Set(defaultGas) }

func (self *ethApi) DefaultGasPrice() *big.Int {
	return self.gpo.SuggestPrice()
}

func (self *ethApi) CurrentBlock() *types.Block {
	return self.eth.BlockChain().CurrentBlock()
}

func (self *ethApi) getBlockByHeight(height int64) *types.Block {
	var num uint64

	switch height {
	case -2:
		return self.eth.Miner().PendingBlock()
	case -1:
		return self.CurrentBlock()
	default:
		if height < 0 {
			return nil
		}

		num = uint64(height)
	}

	return self.eth.BlockChain().GetBlockByNumber(num)
}

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

func isAddress(addr string) bool {
	return addrReg.MatchString(addr)
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                 { return m.from.Nonce() }
func (m callmsg) To() *common.Address           { return m.to }
func (m callmsg) GasPrice() *big.Int            { return m.gasPrice }
func (m callmsg) Gas() *big.Int                 { return m.gas }
func (m callmsg) Value() *big.Int               { return m.value }
func (m callmsg) Data() []byte                  { return m.data }
