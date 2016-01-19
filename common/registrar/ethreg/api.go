// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethreg

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// registryAPIBackend is a backend for an Ethereum Registry.
type registryAPIBackend struct {
	bc      *core.BlockChain
	chainDb ethdb.Database
	txPool  *core.TxPool
	am      *accounts.Manager
}

// PrivateRegistarAPI offers various functions to access the Ethereum registry.
type PrivateRegistarAPI struct {
	be *registryAPIBackend
}

// NewPrivateRegistarAPI creates a new PrivateRegistarAPI instance.
func NewPrivateRegistarAPI(bc *core.BlockChain, chainDb ethdb.Database, txPool *core.TxPool, am *accounts.Manager) *PrivateRegistarAPI {
	return &PrivateRegistarAPI{&registryAPIBackend{bc, chainDb, txPool, am}}
}

// SetGlobalRegistrar allows clients to set the global registry for the node.
// This method can be used to deploy a new registry. First zero out the current
// address by calling the method with namereg = '0x0' and then call this method
// again with '' as namereg. This will submit a transaction to the network which
// will deploy a new registry on execution. The TX hash is returned. When called
// with namereg '' and the current address is not zero the current global is
// address is returned..
func (api *PrivateRegistarAPI) SetGlobalRegistrar(namereg string, from common.Address) (string, error) {
	return registrar.New(api.be).SetGlobalRegistrar(namereg, from)
}

// SetHashReg queries the registry for a hash.
func (api *PrivateRegistarAPI) SetHashReg(hashreg string, from common.Address) (string, error) {
	return registrar.New(api.be).SetHashReg(hashreg, from)
}

// SetUrlHint queries the registry for an url.
func (api *PrivateRegistarAPI) SetUrlHint(hashreg string, from common.Address) (string, error) {
	return registrar.New(api.be).SetUrlHint(hashreg, from)
}

// SaveInfo stores contract information on the local file system.
func (api *PrivateRegistarAPI) SaveInfo(info *compiler.ContractInfo, filename string) (contenthash common.Hash, err error) {
	return compiler.SaveInfo(info, filename)
}

// Register registers a new content hash in the registry.
func (api *PrivateRegistarAPI) Register(sender common.Address, addr common.Address, contentHashHex string) (bool, error) {
	block := api.be.bc.CurrentBlock()
	state, err := state.New(block.Root(), api.be.chainDb)
	if err != nil {
		return false, err
	}

	codeb := state.GetCode(addr)
	codeHash := common.BytesToHash(crypto.Sha3(codeb))
	contentHash := common.HexToHash(contentHashHex)

	_, err = registrar.New(api.be).SetHashToHash(sender, codeHash, contentHash)
	return err == nil, err
}

// RegisterUrl registers a new url in the registry.
func (api *PrivateRegistarAPI) RegisterUrl(sender common.Address, contentHashHex string, url string) (bool, error) {
	_, err := registrar.New(api.be).SetUrlToHash(sender, common.HexToHash(contentHashHex), url)
	return err == nil, err
}

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) {
	return m.from.Address(), nil
}
func (m callmsg) FromFrontier() (common.Address, error) {
	return m.from.Address(), nil
}
func (m callmsg) Nonce() uint64 {
	return m.from.Nonce()
}
func (m callmsg) To() *common.Address {
	return m.to
}
func (m callmsg) GasPrice() *big.Int {
	return m.gasPrice
}
func (m callmsg) Gas() *big.Int {
	return m.gas
}
func (m callmsg) Value() *big.Int {
	return m.value
}
func (m callmsg) Data() []byte {
	return m.data
}

// Call forms a transaction from the given arguments and tries to execute it on
// a private VM with a copy of the state. Any changes are therefore only temporary
// and not part of the actual state. This allows for local execution/queries.
func (be *registryAPIBackend) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, dataStr string) (string, string, error) {
	block := be.bc.CurrentBlock()
	statedb, err := state.New(block.Root(), be.chainDb)
	if err != nil {
		return "", "", err
	}

	var from *state.StateObject
	if len(fromStr) == 0 {
		accounts, err := be.am.Accounts()
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
		msg.gasPrice = new(big.Int).Mul(big.NewInt(50), common.Shannon)
	}

	header := be.bc.CurrentBlock().Header()
	vmenv := core.NewEnv(statedb, be.bc, msg, header)
	gp := new(core.GasPool).AddGas(common.MaxBig)
	res, gas, err := core.ApplyMessage(vmenv, msg, gp)

	return common.ToHex(res), gas.String(), err
}

// StorageAt returns the data stores in the state for the given address and location.
func (be *registryAPIBackend) StorageAt(addr string, storageAddr string) string {
	block := be.bc.CurrentBlock()
	state, err := state.New(block.Root(), be.chainDb)
	if err != nil {
		return ""
	}
	return state.GetState(common.HexToAddress(addr), common.HexToHash(storageAddr)).Hex()
}

// Transact forms a transaction from the given arguments and submits it to the
// transactio pool for execution.
func (be *registryAPIBackend) Transact(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {
	if len(toStr) > 0 && toStr != "0x" && !common.IsHexAddress(toStr) {
		return "", errors.New("invalid address")
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
		gas = big.NewInt(90000)
	} else {
		gas = common.Big(gasStr)
	}

	if len(gasPriceStr) == 0 {
		price = big.NewInt(10000000000000)
	} else {
		price = common.Big(gasPriceStr)
	}

	data = common.FromHex(codeStr)
	if len(toStr) == 0 {
		contractCreation = true
	}

	nonce := be.txPool.State().GetNonce(from)
	if len(nonceStr) != 0 {
		nonce = common.Big(nonceStr).Uint64()
	}

	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreation(nonce, value, gas, price, data)
	} else {
		tx = types.NewTransaction(nonce, to, value, gas, price, data)
	}

	acc := accounts.Account{from}
	signature, err := be.am.Sign(acc, tx.SigHash().Bytes())
	if err != nil {
		return "", err
	}
	signedTx, err := tx.WithSignature(signature)
	if err != nil {
		return "", err
	}

	be.txPool.SetLocal(signedTx)
	if err := be.txPool.Add(signedTx); err != nil {
		return "", nil
	}

	if contractCreation {
		addr := crypto.CreateAddress(from, nonce)
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signedTx.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signedTx.Hash().Hex(), tx.To().Hex())
	}

	return signedTx.Hash().Hex(), nil
}
