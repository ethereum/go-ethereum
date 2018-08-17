// Copyright 2018 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/state"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	defaultMaxMsgSize = 1024 * 1024
	swapProtocolName  = "swap"
	swapVersion       = 1
)

var (
	autoCashInterval     = 300 * time.Second           // default interval for autocash
	autoCashThreshold    = big.NewInt(50000000000000)  // threshold that triggers autocash (wei)
	autoDepositInterval  = 300 * time.Second           // default interval for autocash
	autoDepositThreshold = big.NewInt(50000000000000)  // threshold that triggers autodeposit (wei)
	autoDepositBuffer    = big.NewInt(100000000000000) // buffer that is surplus for fork protection etc (wei)
	buyAt                = big.NewInt(20000000000)     // maximum chunk price host is willing to pay (wei)
	sellAt               = big.NewInt(20000000000)     // minimum chunk price host requires (wei)
	payAt                = big.NewInt(-4096 * 10000)   // threshold that triggers payment {request} (bytes)
	dropAt               = big.NewInt(-4096 * 12000)   // threshold that triggers disconnect (bytes)

	ErrInsufficientFunds = errors.New("Insufficient funds")
	ErrNotAccountedMsg   = errors.New("Message does not need accounting")
)

const (
	chequebookDeployRetries = 5
	chequebookDeployDelay   = 1 * time.Second // delay between retries
)

// SwAP Swarm Accounting Protocol with
//      Swift Automatic  Payments
// a peer to peer micropayment system
type Swap struct {
	chequeManager *ChequeManager
	stateStore    state.Store
	lock          sync.RWMutex
	peers         map[discover.NodeID]*SwapPeer
	local         *Params // local peer's swap parameters
}

type SwapPeer struct {
	*protocols.Peer
	lock        sync.RWMutex
	swapAccount *Swap
	handlerFunc func(context.Context, interface{}) error
	balance     *big.Int
	storeID     string
}

type EntryDirection bool

const (
	DebitEntry  EntryDirection = true
	CreditEntry EntryDirection = false
)

type SwapAccountedMsgType interface {
	GetMsgPrice() *big.Int
}

//Handler for received messages
func (sp *SwapPeer) RunAccountedProtocol(protocolHandler func(ctx context.Context, msg interface{}) error) error {
	//the `peer.Run` function is a loop, so in order to pre-/post-process a message with accounting,
	//we need to save the actual handler
	sp.handlerFunc = protocolHandler

	//then run the handler loop function
	return sp.Run(sp.handleAccountedMsg)
}

//get a peer's balance
func (swap *Swap) GetPeerBalance(peer discover.NodeID) *big.Int {
	if p, ok := swap.peers[peer]; ok {
		return p.balance
	}
	return nil
}

//Handle a received message; this is the handler loop function.
//Check if it needs accounting, and if yes, apply accounting logic:
//Check for sufficient funds, perform operation, then account
func (sp *SwapPeer) handleAccountedMsg(ctx context.Context, msg interface{}) error {
	var err error
	var price *big.Int

	//the message is one which needs accounting...
	if _, ok := msg.(SwapAccountedMsgType); ok {
		//..so first check if there are enough funds for the operation available
		//(for crediting, this means if we are not essentially "overdrafting", or crossing the threshold)
		price, err = sp.checkAvailableFunds(ctx, msg, CreditEntry)
		//if not (or some other error occured), return error
		if err != nil {
			//also, if the error is indeed insufficient funds, then disconnect the peer
			if err == ErrInsufficientFunds {
				log.Error("Insufficient funds, dropping peer")
				sp.Drop(err)
			}
			return err
		}
		//at this point we know there are sufficient funds, so process the message
		err = sp.handlerFunc(ctx, msg)
		if err == nil {
			//and if no errors occurred, finally book the entry
			sp.AccountMsgForPeer(ctx, msg, price, CreditEntry)
		}
	} else {
		//this message doesn't need accounting, so just process it
		err = sp.handlerFunc(ctx, msg)
	}
	return err
}

//Send a message
//Check if it needs accounting, and if yes, apply accounting logic:
//Check for sufficient funds, perform operation, then account
func (sp *SwapPeer) Send(ctx context.Context, msg interface{}) error {
	var err error
	var price *big.Int

	//the message is one which needs accounting...
	if _, ok := msg.(SwapAccountedMsgType); ok {
		//..so first check if there are enough funds for the operation available
		price, err = sp.checkAvailableFunds(ctx, msg, DebitEntry)
		//if not (or some other error occured), return error
		if err != nil {
			//also, if the error is indeed insufficient funds, then disconnect the peer
			if err == ErrInsufficientFunds {
				log.Error("Insufficient funds, dropping peer")
				sp.Drop(err)
			}
			return err
		}
		//at this point we know there are sufficient funds, so process the message
		err = sp.Peer.Send(ctx, msg)
		if err == nil {
			//and if no errors occurred, finally book the entry
			sp.AccountMsgForPeer(ctx, msg, price, DebitEntry)
		}
	} else {
		//this message doesn't need accounting, so just process it
		err = sp.Peer.Send(ctx, msg)
	}
	return err
}

//check that the operation has enough funds available
func (sp *SwapPeer) checkAvailableFunds(ctx context.Context, msg interface{}, direction EntryDirection) (*big.Int, error) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	if accounted, ok := msg.(SwapAccountedMsgType); ok {
		price := accounted.GetMsgPrice()
		//local node is being credited (in its favor), so check upper limit
		if direction == CreditEntry {
			//TODO: is there are a check needed here?
			//It should actually have been done on the client side, the debitor!
			//creditor could theoretically go over payAt, but if well done,
			//should have been checked on the client side so this shouldn't happen?
			checkBalance := sp.balance.Add(sp.balance, price)
			//(checkBalance *Int) Cmp(payAt)
			// -1 if checkBalance  <  payAt
			//  0 if checkBalance  == payAt
			// +1 if checkBalance  >  payAt
			if checkBalance.Cmp(payAt) == 1 {
				return nil, ErrInsufficientFunds
			}
		} else if direction == DebitEntry {
			//NOTE: ErrInsufficientFunds should only be returned
			//if the dropAt is exceeded, but should be ignored for payAt,
			//as there is a "clemency" margin between triggering the check
			//and actually disconnecting the peer

			//(checkBalance *Int) Cmp(dropAt)
			// -1 if checkBalance  <  dropAt
			//  0 if checkBalance  ==	dropAt
			// +1 if checkBalance  >  dropAt
			checkBalance := sp.balance.Sub(sp.balance, price.Abs(price))
			if checkBalance.Cmp(dropAt) == -1 {
				return nil, ErrInsufficientFunds
			}
		}
		return price, nil
	}
	return nil, ErrNotAccountedMsg
}

//The balance is accounted from the point of view of the local node
//Thus, we credit the balance and increase it when the amount is in favor of the local node
//We debit the balance and decrease it when the amount is in favor of the remote peer
func (sp *SwapPeer) AccountMsgForPeer(ctx context.Context, msg interface{}, price *big.Int, direction EntryDirection) {
	if _, ok := msg.(SwapAccountedMsgType); ok {
		sp.lock.Lock()
		defer sp.lock.Unlock()
		//local node is being credited (in its favor), so its balance increases
		if direction == CreditEntry {
			//NOTE: do we need to check for sufficient funds again?
			//operations are not atomic/transactional, so balance may have changed in the meanwhile!
			sp.balance = sp.balance.Add(sp.balance, price)
			//local node is being debited (in favor of remote peer), so its balance decreases
		} else if direction == DebitEntry {
			sp.balance = sp.balance.Sub(sp.balance, price)
		}
		//TODO: save to store here? init store?
		sp.swapAccount.stateStore.Put(sp.storeID, sp.balance)
		//(sp.balance *Int) Cmp(payAt)
		// -1 if sp.balance <  payAt
		//  0 if sp.balance == payAt
		// +1 if sp.balance >  payAt
		if sp.balance.Cmp(payAt) == -1 {
			err := sp.issueCheque(ctx)
			if err != nil {
				//TODO: special error handling, as at this point the accounting has been done
				//but the cheque could not be sent?
			}
		}
		if sp.balance.Cmp(dropAt) == -1 {
			sp.Drop(ErrInsufficientFunds)
		}
		log.Debug(fmt.Sprintf("balance for peer %s: %s", sp.ID(), sp.balance.String()))
	}
}

func (sp *SwapPeer) issueCheque(ctx context.Context) error {
	amount := big.NewInt(0)
	cheque := sp.swapAccount.chequeManager.CreateCheque(sp.ID(), amount.Abs(payAt))
	msg := IssueChequeMsg{
		Cheque: cheque,
	}
	return sp.Send(ctx, msg)
}

//Create a new swap accounted peer
func NewSwapPeer(peer *protocols.Peer, swap *Swap) *SwapPeer {
	balance := big.NewInt(0)
	//check if there is one already in the stateStore and load it
	swap.stateStore.Get(peer.String()[:24]+"-swap", &balance)
	sp := &SwapPeer{
		Peer:        peer,
		swapAccount: swap,
		balance:     balance,
		storeID:     peer.String()[:24] + "-swap",
	}
	swap.lock.Lock()
	defer swap.lock.Unlock()
	swap.peers[peer.ID()] = sp
	return sp
}

// Profile - public swap profile
// public parameters for SWAP, serializable config struct passed in handshake
type Profile struct {
	BuyAt  *big.Int // accepted max price for chunk
	SellAt *big.Int // offered sale price for chunk
	PayAt  *big.Int // threshold that triggers payment request
	DropAt *big.Int // threshold that triggers disconnect
}

// Strategy encapsulates parameters relating to
// automatic deposit and automatic cashing
type Strategy struct {
	AutoCashInterval     time.Duration // default interval for autocash
	AutoCashThreshold    *big.Int      // threshold that triggers autocash (wei)
	AutoDepositInterval  time.Duration // default interval for autocash
	AutoDepositThreshold *big.Int      // threshold that triggers autodeposit (wei)
	AutoDepositBuffer    *big.Int      // buffer that is surplus for fork protection etc (wei)
}

// SwapMsg encapsulates messages transported over pss.
type SwapMsg struct {
	To      []byte
	Control []byte
	Expire  uint32
	Payload *whisper.Envelope
}

// Params extends the public profile with private parameters relating to
// automatic deposit and automatic cashing
type Params struct {
	*Profile
	*Strategy
}

// LocalProfile combines a PayProfile with *swap.Params
type LocalProfile struct {
	*Params
	*PayProfile
}

// RemoteProfile combines a PayProfile with *swap.Profile
type RemoteProfile struct {
	*Profile
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

// New - swap constructor
func NewSwap(local *Params, stateStore state.Store) (swap *Swap, err error) {

	swap = &Swap{
		chequeManager: NewChequeManager(stateStore),
		local:         local,
		stateStore:    stateStore,
		peers:         make(map[discover.NodeID]*SwapPeer),
	}

	//swap.SetParams(local)
	return
}

// NewDefaultSwapParams create params with default values
func NewDefaultSwapParams() *LocalProfile {
	return &LocalProfile{
		PayProfile: &PayProfile{},
		Params: &Params{
			Profile: &Profile{
				BuyAt:  buyAt,
				SellAt: sellAt,
				PayAt:  payAt,
				DropAt: dropAt,
			},
			Strategy: &Strategy{
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

/*
// Add (n)
// n > 0 called when promised/provided n units of service
// n < 0 called when used/requested n units of service
func (swap *Swap) Add(n int) error {
	//defer swap.lock.Unlock()
	//swap.lock.Lock()
	swap.balance += n
	if !swap.Sells && swap.balance > 0 {
		log.Trace(fmt.Sprintf("<%v> remote peer cannot have debt (balance: %v)", swap.proto, swap.balance))
		swap.proto.Drop()
		return fmt.Errorf("[SWAP] <%v> remote peer cannot have debt (balance: %v)", swap.proto, swap.balance)
	}
	if !swap.Buys && swap.balance < 0 {
		log.Trace(fmt.Sprintf("<%v> we cannot have debt (balance: %v)", swap.proto, swap.balance))
		return fmt.Errorf("[SWAP] <%v> we cannot have debt (balance: %v)", swap.proto, swap.balance)
	}
	if swap.balance >= int(swap.local.DropAt) {
		log.Trace(fmt.Sprintf("<%v> remote peer has too much debt (balance: %v, disconnect threshold: %v)", swap.proto, swap.balance, swap.local.DropAt))
		swap.proto.Drop()
		return fmt.Errorf("[SWAP] <%v> remote peer has too much debt (balance: %v, disconnect threshold: %v)", swap.proto, swap.balance, swap.local.DropAt)
	} else if swap.balance <= -int(swap.remote.PayAt) {
		swap.send()
	}
	return nil
}

// Balance accessor
func (swap *Swap) Balance() int {
	//defer swap.lock.Unlock()
	//swap.lock.Lock()
	return swap.balance
}

/*
// send (units) is called when payment is due
// In case of insolvency no promise is issued and sent, safe against fraud
// No return value: no error = payment is opportunistic = hang in till dropped
func (swap *Swap) send() {
	if swap.local.BuyAt != nil && swap.balance < 0 {
		amount := big.NewInt(int64(-swap.balance))
		amount.Mul(amount, swap.remote.SellAt)
		promise, err := swap.Out.Issue(amount)
		if err != nil {
			log.Warn(fmt.Sprintf("<%v> cannot issue cheque (amount: %v, channel: %v): %v", swap.proto, amount, swap.Out, err))
		} else {
			log.Warn(fmt.Sprintf("<%v> cheque issued (amount: %v, channel: %v)", swap.proto, amount, swap.Out))
			swap.proto.Pay(-swap.balance, promise)
			swap.balance = 0
		}
	}
}

// Receive (units, promise) is called by the protocol when a payment msg is received
// returns error if promise is invalid.
func (swap *Swap) Receive(units int, promise Promise) error {
	if units <= 0 {
		return fmt.Errorf("invalid units: %v <= 0", units)
	}

	price := new(big.Int).SetInt64(int64(units))
	price.Mul(price, swap.local.SellAt)

	amount, err := swap.In.Receive(promise)

	if err != nil {
		err = fmt.Errorf("invalid promise: %v", err)
	} else if price.Cmp(amount) != 0 {
		// verify amount = units * unit sale price
		return fmt.Errorf("invalid amount: %v = %v * %v (units sent in msg * agreed sale unit price) != %v (signed in cheque)", price, units, swap.local.SellAt, amount)
	}
	if err != nil {
		log.Trace(fmt.Sprintf("<%v> invalid promise (amount: %v, channel: %v): %v", swap.proto, amount, swap.In, err))
		return err
	}

	// credit remote peer with units
	swap.Add(-units)
	log.Trace(fmt.Sprintf("<%v> received promise (amount: %v, channel: %v): %v", swap.proto, amount, swap.In, promise))

	return nil
}
*/
