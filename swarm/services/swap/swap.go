// Copyright 2016 The go-ethereum Authors
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

package swap

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/chequebook"
	"github.com/ethereum/go-ethereum/contracts/chequebook/contract"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/services/swap/swap"
)

// SwAP       Swarm Accounting Protocol with
// SWAP^2     Strategies of Withholding Automatic Payments
// SWAP^3     Accreditation: payment via credit SWAP
// using chequebook pkg for delayed payments
// default parameters

var (
	autoCashInterval     = 300 * time.Second           // default interval for autocash
	autoCashThreshold    = big.NewInt(50000000000000)  // threshold that triggers autocash (wei)
	autoDepositInterval  = 300 * time.Second           // default interval for autocash
	autoDepositThreshold = big.NewInt(50000000000000)  // threshold that triggers autodeposit (wei)
	autoDepositBuffer    = big.NewInt(100000000000000) // buffer that is surplus for fork protection etc (wei)
	buyAt                = big.NewInt(20000000000)     // maximum chunk price host is willing to pay (wei)
	sellAt               = big.NewInt(20000000000)     // minimum chunk price host requires (wei)
	payAt                = 100                         // threshold that triggers payment {request} (units)
	dropAt               = 10000                       // threshold that triggers disconnect (units)
)

const (
	chequebookDeployRetries = 5
	chequebookDeployDelay   = 1 * time.Second // delay between retries
)

type SwapParams struct {
	*swap.Params
	*PayProfile
}

type SwapProfile struct {
	*swap.Profile
	*PayProfile
}

type PayProfile struct {
	PublicKey   string         // check against signature of promise
	Contract    common.Address // address of chequebook contract
	Beneficiary common.Address // recipient address for swarm sales revenue
	privateKey  *ecdsa.PrivateKey
	publicKey   *ecdsa.PublicKey
	owner       common.Address
	chbook      *chequebook.Chequebook
	lock        sync.RWMutex
}

//create params with default values
func NewDefaultSwapParams() *SwapParams {
	return &SwapParams{
		PayProfile: &PayProfile{},
		Params: &swap.Params{
			Profile: &swap.Profile{
				BuyAt:  buyAt,
				SellAt: sellAt,
				PayAt:  uint(payAt),
				DropAt: uint(dropAt),
			},
			Strategy: &swap.Strategy{
				AutoCashInterval:     autoCashInterval,
				AutoCashThreshold:    autoCashThreshold,
				AutoDepositInterval:  autoDepositInterval,
				AutoDepositThreshold: autoDepositThreshold,
				AutoDepositBuffer:    autoDepositBuffer,
			},
		},
	}
}

//this can only finally be set after all config options (file, cmd line, env vars)
//have been evaluated
func (self *SwapParams) Init(contract common.Address, prvkey *ecdsa.PrivateKey) {
	pubkey := &prvkey.PublicKey

	self.PayProfile = &PayProfile{
		PublicKey:   common.ToHex(crypto.FromECDSAPub(pubkey)),
		Contract:    contract,
		Beneficiary: crypto.PubkeyToAddress(*pubkey),
		privateKey:  prvkey,
		publicKey:   pubkey,
		owner:       crypto.PubkeyToAddress(*pubkey),
	}
}

// swap constructor, parameters
// * global chequebook, assume deployed service and
// * the balance is at buffer.
// swap.Add(n) called in netstore
// n > 0 called when sending chunks = receiving retrieve requests
//                 OR sending cheques.
// n < 0  called when receiving chunks = receiving delivery responses
//                 OR receiving cheques.

func NewSwap(local *SwapParams, remote *SwapProfile, backend chequebook.Backend, proto swap.Protocol) (self *swap.Swap, err error) {
	var (
		ctx = context.TODO()
		ok  bool
		in  *chequebook.Inbox
		out *chequebook.Outbox
	)

	// check if remote chequebook is valid
	// insolvent chequebooks suicide so will signal as invalid
	// TODO: monitoring a chequebooks events
	ok, err = chequebook.ValidateCode(ctx, backend, remote.Contract)
	if !ok {
		log.Info(fmt.Sprintf("invalid contract %v for peer %v: %v)", remote.Contract.Hex()[:8], proto, err))
	} else {
		// remote contract valid, create inbox
		in, err = chequebook.NewInbox(local.privateKey, remote.Contract, local.Beneficiary, crypto.ToECDSAPub(common.FromHex(remote.PublicKey)), backend)
		if err != nil {
			log.Warn(fmt.Sprintf("unable to set up inbox for chequebook contract %v for peer %v: %v)", remote.Contract.Hex()[:8], proto, err))
		}
	}

	// check if local chequebook contract is valid
	ok, err = chequebook.ValidateCode(ctx, backend, local.Contract)
	if !ok {
		log.Warn(fmt.Sprintf("unable to set up outbox for peer %v:  chequebook contract (owner: %v): %v)", proto, local.owner.Hex(), err))
	} else {
		out = chequebook.NewOutbox(local.Chequebook(), remote.Beneficiary)
	}

	pm := swap.Payment{
		In:    in,
		Out:   out,
		Buys:  out != nil,
		Sells: in != nil,
	}
	self, err = swap.New(local.Params, pm, proto)
	if err != nil {
		return
	}
	// remote profile given (first) in handshake
	self.SetRemote(remote.Profile)
	var buy, sell string
	if self.Buys {
		buy = "purchase from peer enabled at " + remote.SellAt.String() + " wei/chunk"
	} else {
		buy = "purchase from peer disabled"
	}
	if self.Sells {
		sell = "selling to peer enabled at " + local.SellAt.String() + " wei/chunk"
	} else {
		sell = "selling to peer disabled"
	}
	log.Warn(fmt.Sprintf("SWAP arrangement with <%v>: %v; %v)", proto, buy, sell))

	return
}

func (self *SwapParams) Chequebook() *chequebook.Chequebook {
	defer self.lock.Unlock()
	self.lock.Lock()
	return self.chbook
}

func (self *SwapParams) PrivateKey() *ecdsa.PrivateKey {
	return self.privateKey
}

// func (self *SwapParams) PublicKey() *ecdsa.PublicKey {
// 	return self.publicKey
// }

func (self *SwapParams) SetKey(prvkey *ecdsa.PrivateKey) {
	self.privateKey = prvkey
	self.publicKey = &prvkey.PublicKey
}

// setChequebook(path, backend) wraps the
// chequebook initialiser and sets up autoDeposit to cover spending.
func (self *SwapParams) SetChequebook(ctx context.Context, backend chequebook.Backend, path string) error {
	self.lock.Lock()
	contract := self.Contract
	self.lock.Unlock()

	valid, err := chequebook.ValidateCode(ctx, backend, contract)
	if err != nil {
		return err
	} else if valid {
		return self.newChequebookFromContract(path, backend)
	}
	return self.deployChequebook(ctx, backend, path)
}

func (self *SwapParams) deployChequebook(ctx context.Context, backend chequebook.Backend, path string) error {
	opts := bind.NewKeyedTransactor(self.privateKey)
	opts.Value = self.AutoDepositBuffer
	opts.Context = ctx

	log.Info(fmt.Sprintf("Deploying new chequebook (owner: %v)", opts.From.Hex()))
	contract, err := deployChequebookLoop(opts, backend)
	if err != nil {
		log.Error(fmt.Sprintf("unable to deploy new chequebook: %v", err))
		return err
	}
	log.Info(fmt.Sprintf("new chequebook deployed at %v (owner: %v)", contract.Hex(), opts.From.Hex()))

	// need to save config at this point
	self.lock.Lock()
	self.Contract = contract
	err = self.newChequebookFromContract(path, backend)
	self.lock.Unlock()
	if err != nil {
		log.Warn(fmt.Sprintf("error initialising cheque book (owner: %v): %v", opts.From.Hex(), err))
	}
	return err
}

// repeatedly tries to deploy a chequebook.
func deployChequebookLoop(opts *bind.TransactOpts, backend chequebook.Backend) (addr common.Address, err error) {
	var tx *types.Transaction
	for try := 0; try < chequebookDeployRetries; try++ {
		if try > 0 {
			time.Sleep(chequebookDeployDelay)
		}
		if _, tx, _, err = contract.DeployChequebook(opts, backend); err != nil {
			log.Warn(fmt.Sprintf("can't send chequebook deploy tx (try %d): %v", try, err))
			continue
		}
		if addr, err = bind.WaitDeployed(opts.Context, backend, tx); err != nil {
			log.Warn(fmt.Sprintf("chequebook deploy error (try %d): %v", try, err))
			continue
		}
		return addr, nil
	}
	return addr, err
}

// initialise the chequebook from a persisted json file or create a new one
// caller holds the lock
func (self *SwapParams) newChequebookFromContract(path string, backend chequebook.Backend) error {
	hexkey := common.Bytes2Hex(self.Contract.Bytes())
	err := os.MkdirAll(filepath.Join(path, "chequebooks"), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directory for chequebooks: %v", err)
	}

	chbookpath := filepath.Join(path, "chequebooks", hexkey+".json")
	self.chbook, err = chequebook.LoadChequebook(chbookpath, self.privateKey, backend, true)

	if err != nil {
		self.chbook, err = chequebook.NewChequebook(chbookpath, self.Contract, self.privateKey, backend)
		if err != nil {
			log.Warn(fmt.Sprintf("unable to initialise chequebook (owner: %v): %v", self.owner.Hex(), err))
			return fmt.Errorf("unable to initialise chequebook (owner: %v): %v", self.owner.Hex(), err)
		}
	}

	self.chbook.AutoDeposit(self.AutoDepositInterval, self.AutoDepositThreshold, self.AutoDepositBuffer)
	log.Info(fmt.Sprintf("auto deposit ON for %v -> %v: interval = %v, threshold = %v, buffer = %v)", crypto.PubkeyToAddress(*(self.publicKey)).Hex()[:8], self.Contract.Hex()[:8], self.AutoDepositInterval, self.AutoDepositThreshold, self.AutoDepositBuffer))

	return nil
}
