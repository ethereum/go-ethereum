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
	"errors"
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
	"github.com/ethereum/go-ethereum/swarm/log"
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

// LocalProfile combines a PayProfile with *swap.Params
type LocalProfile struct {
	*swap.Params
	*PayProfile
}

// RemoteProfile combines a PayProfile with *swap.Profile
type RemoteProfile struct {
	*swap.Profile
	*PayProfile
}

// PayProfile is a container for relevant chequebook and beneficiary options
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

// NewDefaultSwapParams create params with default values
func NewDefaultSwapParams() *LocalProfile {
	return &LocalProfile{
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

// Init this can only finally be set after all config options (file, cmd line, env vars)
// have been evaluated
func (lp *LocalProfile) Init(contract common.Address, prvkey *ecdsa.PrivateKey) {
	pubkey := &prvkey.PublicKey

	lp.PayProfile = &PayProfile{
		PublicKey:   common.ToHex(crypto.FromECDSAPub(pubkey)),
		Contract:    contract,
		Beneficiary: crypto.PubkeyToAddress(*pubkey),
		privateKey:  prvkey,
		publicKey:   pubkey,
		owner:       crypto.PubkeyToAddress(*pubkey),
	}
}

// NewSwap constructor, parameters
// * global chequebook, assume deployed service and
// * the balance is at buffer.
// swap.Add(n) called in netstore
// n > 0 called when sending chunks = receiving retrieve requests
//                 OR sending cheques.
// n < 0  called when receiving chunks = receiving delivery responses
//                 OR receiving cheques.
func NewSwap(localProfile *LocalProfile, remoteProfile *RemoteProfile, backend chequebook.Backend, proto swap.Protocol) (swapInstance *swap.Swap, err error) {
	var (
		ctx = context.TODO()
		ok  bool
		in  *chequebook.Inbox
		out *chequebook.Outbox
	)

	remotekey, err := crypto.UnmarshalPubkey(common.FromHex(remoteProfile.PublicKey))
	if err != nil {
		return nil, errors.New("invalid remote public key")
	}

	// check if remoteProfile chequebook is valid
	// insolvent chequebooks suicide so will signal as invalid
	// TODO: monitoring a chequebooks events
	ok, err = chequebook.ValidateCode(ctx, backend, remoteProfile.Contract)
	if !ok {
		log.Info(fmt.Sprintf("invalid contract %v for peer %v: %v)", remoteProfile.Contract.Hex()[:8], proto, err))
	} else {
		// remoteProfile contract valid, create inbox
		in, err = chequebook.NewInbox(localProfile.privateKey, remoteProfile.Contract, localProfile.Beneficiary, remotekey, backend)
		if err != nil {
			log.Warn(fmt.Sprintf("unable to set up inbox for chequebook contract %v for peer %v: %v)", remoteProfile.Contract.Hex()[:8], proto, err))
		}
	}

	// check if localProfile chequebook contract is valid
	ok, err = chequebook.ValidateCode(ctx, backend, localProfile.Contract)
	if !ok {
		log.Warn(fmt.Sprintf("unable to set up outbox for peer %v:  chequebook contract (owner: %v): %v)", proto, localProfile.owner.Hex(), err))
	} else {
		out = chequebook.NewOutbox(localProfile.Chequebook(), remoteProfile.Beneficiary)
	}

	pm := swap.Payment{
		In:    in,
		Out:   out,
		Buys:  out != nil,
		Sells: in != nil,
	}
	swapInstance, err = swap.New(localProfile.Params, pm, proto)
	if err != nil {
		return
	}
	// remoteProfile profile given (first) in handshake
	swapInstance.SetRemote(remoteProfile.Profile)
	var buy, sell string
	if swapInstance.Buys {
		buy = "purchase from peer enabled at " + remoteProfile.SellAt.String() + " wei/chunk"
	} else {
		buy = "purchase from peer disabled"
	}
	if swapInstance.Sells {
		sell = "selling to peer enabled at " + localProfile.SellAt.String() + " wei/chunk"
	} else {
		sell = "selling to peer disabled"
	}
	log.Warn(fmt.Sprintf("SWAP arrangement with <%v>: %v; %v)", proto, buy, sell))

	return
}

// Chequebook get's chequebook from the localProfile
func (lp *LocalProfile) Chequebook() *chequebook.Chequebook {
	defer lp.lock.Unlock()
	lp.lock.Lock()
	return lp.chbook
}

// PrivateKey accessor
func (lp *LocalProfile) PrivateKey() *ecdsa.PrivateKey {
	return lp.privateKey
}

// func (self *LocalProfile) PublicKey() *ecdsa.PublicKey {
// 	return self.publicKey
// }

// SetKey set's private and public key on localProfile
func (lp *LocalProfile) SetKey(prvkey *ecdsa.PrivateKey) {
	lp.privateKey = prvkey
	lp.publicKey = &prvkey.PublicKey
}

// SetChequebook wraps the chequebook initialiser and sets up autoDeposit to cover spending.
func (lp *LocalProfile) SetChequebook(ctx context.Context, backend chequebook.Backend, path string) error {
	lp.lock.Lock()
	swapContract := lp.Contract
	lp.lock.Unlock()

	valid, err := chequebook.ValidateCode(ctx, backend, swapContract)
	if err != nil {
		return err
	} else if valid {
		return lp.newChequebookFromContract(path, backend)
	}
	return lp.deployChequebook(ctx, backend, path)
}

// deployChequebook deploys the localProfile Chequebook
func (lp *LocalProfile) deployChequebook(ctx context.Context, backend chequebook.Backend, path string) error {
	opts := bind.NewKeyedTransactor(lp.privateKey)
	opts.Value = lp.AutoDepositBuffer
	opts.Context = ctx

	log.Info(fmt.Sprintf("Deploying new chequebook (owner: %v)", opts.From.Hex()))
	address, err := deployChequebookLoop(opts, backend)
	if err != nil {
		log.Error(fmt.Sprintf("unable to deploy new chequebook: %v", err))
		return err
	}
	log.Info(fmt.Sprintf("new chequebook deployed at %v (owner: %v)", address.Hex(), opts.From.Hex()))

	// need to save config at this point
	lp.lock.Lock()
	lp.Contract = address
	err = lp.newChequebookFromContract(path, backend)
	lp.lock.Unlock()
	if err != nil {
		log.Warn(fmt.Sprintf("error initialising cheque book (owner: %v): %v", opts.From.Hex(), err))
	}
	return err
}

// deployChequebookLoop repeatedly tries to deploy a chequebook.
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

// newChequebookFromContract - initialise the chequebook from a persisted json file or create a new one
// caller holds the lock
func (lp *LocalProfile) newChequebookFromContract(path string, backend chequebook.Backend) error {
	hexkey := common.Bytes2Hex(lp.Contract.Bytes())
	err := os.MkdirAll(filepath.Join(path, "chequebooks"), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directory for chequebooks: %v", err)
	}

	chbookpath := filepath.Join(path, "chequebooks", hexkey+".json")
	lp.chbook, err = chequebook.LoadChequebook(chbookpath, lp.privateKey, backend, true)

	if err != nil {
		lp.chbook, err = chequebook.NewChequebook(chbookpath, lp.Contract, lp.privateKey, backend)
		if err != nil {
			log.Warn(fmt.Sprintf("unable to initialise chequebook (owner: %v): %v", lp.owner.Hex(), err))
			return fmt.Errorf("unable to initialise chequebook (owner: %v): %v", lp.owner.Hex(), err)
		}
	}

	lp.chbook.AutoDeposit(lp.AutoDepositInterval, lp.AutoDepositThreshold, lp.AutoDepositBuffer)
	log.Info(fmt.Sprintf("auto deposit ON for %v -> %v: interval = %v, threshold = %v, buffer = %v)", crypto.PubkeyToAddress(*(lp.publicKey)).Hex()[:8], lp.Contract.Hex()[:8], lp.AutoDepositInterval, lp.AutoDepositThreshold, lp.AutoDepositBuffer))

	return nil
}
