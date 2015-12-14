package swap

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/services/chequebook"
	"github.com/ethereum/go-ethereum/swarm/services/chequebook/contract"
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
	// backend     bind.Backend
	lock sync.RWMutex
}

func DefaultSwapParams(contract common.Address, prvkey *ecdsa.PrivateKey) *SwapParams {
	pubkey := &prvkey.PublicKey
	return &SwapParams{
		PayProfile: &PayProfile{
			PublicKey:   common.ToHex(crypto.FromECDSAPub(pubkey)),
			Contract:    contract,
			Beneficiary: crypto.PubkeyToAddress(*pubkey),
			privateKey:  prvkey,
			publicKey:   pubkey,
			owner:       crypto.PubkeyToAddress(*pubkey),
		},
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

// swap constructor, parameters
// * global chequebook, assume deployed service and
// * the balance is at buffer.
// swap.Add(n) called in netstore
// n > 0 called when sending chunks = receiving retrieve requests
//                 OR sending cheques.
// n < 0  called when receiving chunks = receiving delivery responses
//                 OR receiving cheques.

func NewSwap(local *SwapParams, remote *SwapProfile, backend bind.Backend, proto swap.Protocol) (self *swap.Swap, err error) {

	// check if remote chequebook is valid
	// insolvent chequebooks suicide so will signal as invalid
	// TODO: monitoring a chequebooks events
	var in *chequebook.Inbox
	err = bind.Validate(remote.Contract, contract.ContractDeployedCode, backend)
	if err != nil {
		glog.V(logger.Info).Infof("[BZZ] SWAP invalid contract %v for peer %v: %v)", remote.Contract.Hex()[:8], proto, err)
	} else {
		// remote contract valid, create inbox
		in, err = chequebook.NewInbox(local.privateKey, remote.Contract, local.Beneficiary, crypto.ToECDSAPub(common.FromHex(remote.PublicKey)), backend)
		if err != nil {
			glog.V(logger.Warn).Infof("[BZZ] SWAP unable to set up inbox for chequebook contract %v for peer %v: %v)", remote.Contract.Hex()[:8], proto, err)
		}
	}

	// cheque if local chequebook contract is valid
	var out *chequebook.Outbox
	err = bind.Validate(local.Contract, contract.ContractDeployedCode, backend)
	if err != nil {
		glog.V(logger.Warn).Infof("[BZZ] SWAP unable to set up outbox for peer %v:  chequebook contract (owner: %v): %v)", proto, local.owner.Hex(), err)
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
	glog.V(logger.Warn).Infof("[BZZ] SWAP arrangement with <%v>: %v; %v)", proto, buy, sell)

	return
}

func (self *SwapParams) Chequebook() *chequebook.Chequebook {
	defer self.lock.Unlock()
	self.lock.Lock()
	return self.chbook
}

// func (self *SwapParams) Backend() bind.Backend {
// 	return self.backend
// }

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
func (self *SwapParams) SetChequebook(path string, backend bind.Backend) (done chan bool, err error) {
	defer self.lock.Unlock()
	self.lock.Lock()
	var valid bool
	done = make(chan bool)
	err = bind.Validate(self.Contract, contract.ContractDeployedCode, backend)
	if err != nil {
		go self.deployChequebook(path, backend, done)
	} else {
		valid = true
		go func() {
			done <- false
			close(done)
		}()
	}
	if valid {
		err = self.newChequebookFromContract(path, backend)
		return done, err
	}
	return done, nil
}

func deploy(deployTransactor *bind.TransactOpts, backend bind.ContractBackend) (*types.Transaction, error) {
	_, tx, _, err := contract.DeployChequebook(deployTransactor, backend)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// prvKey *ecdsa.PrivateKey, amount *big.Int,

func (self *SwapParams) deployChequebook(path string, backend bind.Backend, done chan bool) {
	owner := crypto.PubkeyToAddress(*(self.publicKey))
	glog.V(logger.Info).Infof("[BZZ] SWAP Deploying new chequebook (owner: %v)", owner.Hex())
	var valid bool

	transactOpts := bind.NewKeyedTransactor(self.privateKey)
	transactOpts.Value = self.AutoDepositBuffer
	contract, err := bind.Deploy(deploy, contract.ContractDeployedCode, transactOpts, bind.DefaultDeployOptions(), backend)
	if err != nil {
		glog.V(logger.Error).Infof("[BZZ] SWAP unable to deploy new chequebook: %v", err)
		return
	}

	// need to save config at this point
	self.lock.Lock()
	self.Contract = contract
	err = self.newChequebookFromContract(path, backend)
	self.lock.Unlock()
	if err != nil {
		glog.V(logger.Warn).Infof("[BZZ] SWAP error initialising cheque book (owner: %v): %v", owner.Hex(), err)
	} else {
		valid = true
		glog.V(logger.Info).Infof("[BZZ] SWAP new chequebook deployed at %v (owner: %v)", contract.Hex(), owner.Hex())
	}
	done <- valid
	close(done)
}

// initialise the chequebook from a persisted json file or create a new one
// caller holds the lock
func (self *SwapParams) newChequebookFromContract(path string, backend bind.Backend) error {
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
			glog.V(logger.Warn).Infof("[BZZ] SWAP unable to initialise chequebook (owner: %v): %v", self.owner.Hex(), err)
			return fmt.Errorf("unable to initialise chequebook (owner: %v): %v", self.owner.Hex(), err)
		}
	}

	self.chbook.AutoDeposit(self.AutoDepositInterval, self.AutoDepositThreshold, self.AutoDepositBuffer)
	glog.V(logger.Info).Infof("[BZZ] SWAP auto deposit ON for %v -> %v: interval = %v, threshold = %v, buffer = %v)", crypto.PubkeyToAddress(*(self.publicKey)).Hex()[:8], self.Contract.Hex()[:8], self.AutoDepositInterval, self.AutoDepositThreshold, self.AutoDepositBuffer)

	return nil
}
