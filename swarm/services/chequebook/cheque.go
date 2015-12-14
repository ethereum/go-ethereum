//go:generate abigen --sol contract/chequebook.sol --pkg contract --out contract/jaak.go
package chequebook

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/services/chequebook/contract"
	"github.com/ethereum/go-ethereum/swarm/services/swap/swap"
)

// func init() {
// 	glog.SetToStderr(true)
// 	glog.SetV(6)
// }

/*
Chequebook package is a go API to the 'chequebook' ethereum smart contract
With convenience methods that allow using chequebook for
* issuing, receiving, verifying cheques in ether
* (auto)cashing cheques in ether
* (auto)depositing ether to the chequebook contract
TODO:
* watch peer solvency and notify of bouncing cheques
* enable paying with cheque by signing off

Some functionality require interacting with the blockchain:
* setting current balance on peer's chequebook
* sending the transaction to cash the cheque
* depositing ether to the chequebook
* watching incoming ether

Backend is the interface for that
*/

var (
	gasToCash = big.NewInt(2000000) // gas cost of a cash transaction using chequebook
	// gasToDeploy = big.NewInt(3000000)
)

// rlp serialised cheque model for use with the chequebook
type Cheque struct {
	// the address of the contract itself needed to avoid cross-contract submission
	Contract    common.Address // contract address
	Beneficiary common.Address // beneficiary
	Amount      *big.Int       // cumulative amount of all funds sent
	Sig         []byte         // signature Sign(Sha3(contract, beneficiary, amount), prvKey)
}

func (self *Cheque) String() string {
	return fmt.Sprintf("contract: %s, beneficiary: %s, amount: %v, signature: %x", self.Contract.Hex(), self.Beneficiary.Hex(), self.Amount, self.Sig)
}

type Params struct {
	ContractCode, ContractAbi string
}

var ContractParams = &Params{contract.ChequebookBin, contract.ChequebookABI}

// chequebook to create, sign cheques from single contract to multiple beneficiarys
// outgoing payment handler for peer to peer micropayments
type Chequebook struct {
	path    string            // path to chequebook file
	prvKey  *ecdsa.PrivateKey // private key to sign cheque with
	lock    sync.Mutex        //
	backend bind.Backend      // blockchain API
	// abigen  bind.ContractBackend // blockchain contract backend for abigen
	quit     chan bool                   // when closed causes autodeposit to stop
	owner    common.Address              // owner address (derived from pubkey)
	contract *contract.Chequebook        // abigen binding
	session  *contract.ChequebookSession // abigen binding with Tx Opts

	// persisted fields
	balance      *big.Int                    // not synced with blockchain
	contractAddr common.Address              // contract address
	sent         map[common.Address]*big.Int //tallies for beneficiarys

	txhash    string   // tx hash of last deposit tx
	threshold *big.Int // threshold that triggers autodeposit if not nil
	buffer    *big.Int // buffer to keep on top of balance for fork protection
}

func (self *Chequebook) String() string {
	return fmt.Sprintf("contract: %s, owner: %s, balance: %v, signer: %x", self.contractAddr.Hex(), self.owner.Hex(), self.balance, self.prvKey.PublicKey)
}

// NewChequebook(path, contract, prvKey, abibbackend, backend) creates a new Chequebook
func NewChequebook(path string, contractAddr common.Address, prvKey *ecdsa.PrivateKey, backend bind.Backend) (self *Chequebook, err error) {
	balance := new(big.Int)
	sent := make(map[common.Address]*big.Int)

	chbook, err := contract.NewChequebook(contractAddr, backend)
	if err != nil {
		return nil, err
	}
	transactOpts := bind.NewKeyedTransactor(prvKey)
	session := &contract.ChequebookSession{
		Contract:     chbook,
		TransactOpts: *transactOpts,
	}

	self = &Chequebook{
		prvKey:       prvKey,
		balance:      balance,
		contractAddr: contractAddr,
		sent:         sent,
		path:         path,
		backend:      backend,
		owner:        transactOpts.From,
		contract:     chbook,
		session:      session,
	}

	if (contractAddr != common.Address{}) {
		self.setBalanceFromBlockChain()
		glog.V(logger.Detail).Infof("[CHEQUEBOOK] new chequebook initialised from %v (owner: %v, balance: %s)", contractAddr, self.owner.Hex(), self.balance.String())
	}
	return
}

func (self *Chequebook) setBalanceFromBlockChain() {
	balance := self.backend.BalanceAt(self.contractAddr)
	self.balance.Set(balance)
}

// LoadChequebook(path, prvKey, backend) loads a chequebook from disk (file path)
func LoadChequebook(path string, prvKey *ecdsa.PrivateKey, backend bind.Backend, checkBalance bool) (self *Chequebook, err error) {
	var data []byte
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}

	self, _ = NewChequebook(path, common.Address{}, prvKey, backend)

	err = json.Unmarshal(data, self)
	if err != nil {
		return nil, err
	}
	if checkBalance {
		self.setBalanceFromBlockChain()
	}

	glog.V(logger.Detail).Infof("[CHEQUEBOOK] loaded chequebook (%s, owner: %v, balance: %v) initialised from %v", self.contractAddr.Hex(), self.owner.Hex(), self.balance, path)

	return
}

// chequebook serialisation
type chequebookFile struct {
	Balance  string
	Contract string
	Owner    string
	Sent     map[string]string
}

func (self *Chequebook) UnmarshalJSON(data []byte) error {
	var file chequebookFile
	err := json.Unmarshal(data, &file)
	if err != nil {
		return err
	}
	_, ok := self.balance.SetString(file.Balance, 10)
	if !ok {
		return fmt.Errorf("cumulative amount sent: unable to convert string to big integer: %v", file.Balance)
	}
	self.contractAddr = common.HexToAddress(file.Contract)
	for addr, sent := range file.Sent {
		self.sent[common.HexToAddress(addr)], ok = new(big.Int).SetString(sent, 10)
		if !ok {
			return fmt.Errorf("beneficiary %v cumulative amount sent: unable to convert string to big integer: %v", addr, sent)
		}
	}
	return nil
}

func (self *Chequebook) MarshalJSON() ([]byte, error) {
	var file = &chequebookFile{
		Balance:  self.balance.String(),
		Contract: self.contractAddr.Hex(),
		Owner:    self.owner.Hex(),
		Sent:     make(map[string]string),
	}
	for addr, sent := range self.sent {
		file.Sent[addr.Hex()] = sent.String()
	}
	return json.Marshal(file)
}

// Save() persists the chequebook on disk
// remembers balance, contract address and
// cumulative amount of funds sent for each beneficiary
func (self *Chequebook) Save() (err error) {
	data, err := json.MarshalIndent(self, "", " ")
	if err != nil {
		return err
	}
	glog.V(logger.Detail).Infof("[CHEQUEBOOK] saving chequebook (%s) to %v", self.contractAddr.Hex(), self.path)

	return ioutil.WriteFile(self.path, data, os.ModePerm)
}

// Stop() quits the autodeposit go routine to terminate
func (self *Chequebook) Stop() {
	defer self.lock.Unlock()
	self.lock.Lock()
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
}

// Issue(beneficiary, amount) will create a Cheque
// the cheque is signed by the chequebook owner's private key
// the signer commits to a contract (one that they own), a beneficiary and amount
func (self *Chequebook) Issue(beneficiary common.Address, amount *big.Int) (ch *Cheque, err error) {
	defer self.lock.Unlock()
	self.lock.Lock()
	glog.V(logger.Detail).Infof("[CHEQUEBOOK] prvKey: %v", self.prvKey)
	if amount.Cmp(common.Big0) <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero (%v)", amount)
	}
	if self.balance.Cmp(amount) < 0 {
		err = fmt.Errorf("insufficent funds to issue cheque for amount: %v. balance: %v", amount, self.balance)
	} else {
		var sig []byte
		sent, found := self.sent[beneficiary]
		if !found {
			sent = new(big.Int)
			self.sent[beneficiary] = sent
		}
		sum := new(big.Int).Set(sent)
		sum.Add(sum, amount)

		sig, err = crypto.Sign(sigHash(self.contractAddr, beneficiary, sum), self.prvKey)
		if err == nil {
			ch = &Cheque{
				Contract:    self.contractAddr,
				Beneficiary: beneficiary,
				Amount:      sum,
				Sig:         sig,
			}
			sent.Set(sum)
			self.balance.Sub(self.balance, amount) // subtract amount from balance
		}
	}

	// auto deposit if threshold is set and balance is less then threshold
	// note this is called even if issueing cheque fails
	// so we reattempt depositing
	if self.threshold != nil {
		if self.balance.Cmp(self.threshold) < 0 {
			send := new(big.Int).Sub(self.buffer, self.balance)
			self.deposit(send)
		}
	}

	return
}

// convenience method to cash any cheque
func (self *Chequebook) Cash(ch *Cheque) (txhash string, err error) {
	return ch.Cash(self.session)
}

// data to sign: contract address, beneficiary, cumulative amount of funds ever sent
func sigHash(contract, beneficiary common.Address, sum *big.Int) []byte {
	bigamount := sum.Bytes()
	if len(bigamount) > 32 {
		return nil
	}
	var amount32 [32]byte
	copy(amount32[32-len(bigamount):32], bigamount)
	input := append(contract.Bytes(), beneficiary.Bytes()...)
	input = append(input, amount32[:]...)
	return crypto.Sha3(input)
}

// Balance() public accessor for balance
func (self *Chequebook) Balance() *big.Int {
	defer self.lock.Unlock()
	self.lock.Lock()
	return new(big.Int).Set(self.balance)
}

// Balance() public accessor for balance
func (self *Chequebook) Owner() common.Address {
	return self.owner
}

// Backend() public accessor for backend
func (self *Chequebook) Backend() bind.Backend {
	return self.backend
}

// Address() public accessor for contract
func (self *Chequebook) Address() common.Address {
	return self.contractAddr
}

// Deposit(amount) deposits amount to the chequebook account
func (self *Chequebook) Deposit(amount *big.Int) (string, error) {
	defer self.lock.Unlock()
	self.lock.Lock()
	return self.deposit(amount)
}

// deposit(amount) deposits amount to the chequebook account
// caller holds the lock
func (self *Chequebook) deposit(amount *big.Int) (string, error) {
	// since the amount is variable here, we do not use sessions
	depositTransactor := bind.NewKeyedTransactor(self.prvKey)
	depositTransactor.Value = amount
	chbookRaw := &contract.ChequebookRaw{self.contract}
	tx, err := chbookRaw.Transfer(depositTransactor)
	// assume that transaction is actually successful, we add the amount to balance right away
	if err != nil {
		glog.V(logger.Warn).Infof("[CHEQUEBOOK] error depositing %d wei to chequebook (%s, balance: %v, target: %v): %v", amount, self.contractAddr.Hex(), self.balance, self.buffer, err)
	} else {
		self.balance.Add(self.balance, amount)
		glog.V(logger.Detail).Infof("[CHEQUEBOOK] deposited %d wei to chequebook (%s, balance: %v, target: %v)", amount, self.contractAddr.Hex(), self.balance, self.buffer)
	}
	return tx.Hash().Hex(), err
}

// AutoDeposit(interval, threshold, buffer) (re)sets interval time and amount
// which triggers sending funds to the chequebook contract
// backend needs to be set
// if threshold is not less than buffer, then deposit will be triggered on
// every new cheque issued
func (self *Chequebook) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	defer self.lock.Unlock()
	self.lock.Lock()
	self.threshold = threshold
	self.buffer = buffer
	self.autoDeposit(interval)
}

// autoDeposit(interval) starts a go routine that periodically sends funds to the
// chequebook contract
// caller holds the lock
// the go routine terminates if Chequebook.quit us closed
func (self *Chequebook) autoDeposit(interval time.Duration) {
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
	// if threshold >= balance autodeposit after every cheque issued
	if interval == time.Duration(0) || self.threshold != nil && self.buffer != nil && self.threshold.Cmp(self.buffer) >= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	self.quit = make(chan bool)
	quit := self.quit
	go func() {
	FOR:
		for {
			select {
			case <-quit:
				break FOR
			case <-ticker.C:
				self.lock.Lock()
				if self.balance.Cmp(self.buffer) < 0 {
					amount := new(big.Int).Sub(self.buffer, self.balance)
					txhash, err := self.deposit(amount)
					if err == nil {
						self.txhash = txhash
					}
				}
				self.lock.Unlock()
			}
		}
	}()
	return
}

type Outbox struct {
	chequeBook  *Chequebook
	beneficiary common.Address
}

func NewOutbox(chbook *Chequebook, beneficiary common.Address) *Outbox {
	return &Outbox{chbook, beneficiary}
}

func (self *Outbox) Issue(amount *big.Int) (swap.Promise, error) {
	return self.chequeBook.Issue(self.beneficiary, amount)
}

func (self *Outbox) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	self.chequeBook.AutoDeposit(interval, threshold, buffer)
}

func (self *Outbox) Stop() {}

func (self *Outbox) String() string {
	return fmt.Sprintf("chequebook: %v, beneficiary: %s, balance: %v", self.chequeBook.Address().Hex(), self.beneficiary.Hex(), self.chequeBook.Balance())
}

// inbox to deposit, verify and cash cheques
// from a single contract to single beneficiary
// incoming payment handler for peer to peer micropayments
type Inbox struct {
	lock        sync.Mutex
	contract    common.Address              // peer's chequebook contract
	beneficiary common.Address              // local peer's receiving address
	sender      common.Address              // local peer's address to send cashing tx from
	signer      *ecdsa.PublicKey            // peer's public key
	txhash      string                      // tx hash of last cashing tx
	abigen      bind.ContractBackend        // blockchain API
	session     *contract.ChequebookSession // abi contract backend with tx opts
	quit        chan bool                   // when closed causes autocash to stop
	maxUncashed *big.Int                    // threshold that triggers autocashing
	cashed      *big.Int                    // cumulative amount cashed
	cheque      *Cheque                     // last cheque, nil if none yet received
}

// NewInbox(contract, beneficiary, signer, backend) constructor for Inbox
// not persisted, cumulative sum updated from blockchain when first cheque received
// backend used to sync amount (Call) as well as cash the cheques (Transact)
func NewInbox(prvKey *ecdsa.PrivateKey, contractAddr, beneficiary common.Address, signer *ecdsa.PublicKey, abigen bind.ContractBackend) (self *Inbox, err error) {

	if signer == nil {
		return nil, fmt.Errorf("signer is null")
	}
	chbook, err := contract.NewChequebook(contractAddr, abigen)
	if err != nil {
		return nil, err
	}
	transactOpts := bind.NewKeyedTransactor(prvKey)
	transactOpts.GasLimit = gasToCash
	session := &contract.ChequebookSession{
		Contract:     chbook,
		TransactOpts: *transactOpts,
	}
	sender := transactOpts.From

	self = &Inbox{
		contract:    contractAddr,
		beneficiary: beneficiary,
		sender:      sender,
		signer:      signer,
		session:     session,
		cashed:      new(big.Int).Set(common.Big0),
	}
	glog.V(logger.Detail).Infof("[CHEQUEBOOK] initialised inbox (%s -> %s) expected signer: %x", self.contract.Hex(), self.beneficiary.Hex(), crypto.FromECDSAPub(signer))
	return
}

func (self *Inbox) String() string {
	return fmt.Sprintf("chequebook: %v, beneficiary: %s, balance: %v", self.contract.Hex(), self.beneficiary.Hex(), self.cheque.Amount)
}

// Stop() quits the autocash go routine to terminate
func (self *Inbox) Stop() {
	defer self.lock.Unlock()
	self.lock.Lock()
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
}

func (self *Inbox) Cash() (txhash string, err error) {
	if self.cheque != nil {
		txhash, err = self.cheque.Cash(self.session)
		glog.V(logger.Detail).Infof("[CHEQUEBOOK] cashing cheque (total: %v) on chequebook (%s) sending to %v", self.cheque.Amount, self.contract.Hex(), self.beneficiary.Hex())
		self.cashed = self.cheque.Amount
	}
	return
}

// AutoCash(cashInterval, maxUncashed) (re)sets maximum time and amount which
// triggers cashing of the last uncashed cheque
// if maxUncashed is set to 0, then autocash on receipt
func (self *Inbox) AutoCash(cashInterval time.Duration, maxUncashed *big.Int) {
	defer self.lock.Unlock()
	self.lock.Lock()
	self.maxUncashed = maxUncashed
	self.autoCash(cashInterval)
}

// autoCash(d) starts a loop that periodically clears the last check
// if the peer is trusted, clearing period could be 24h, or a week
// caller holds the lock
func (self *Inbox) autoCash(cashInterval time.Duration) {
	if self.quit != nil {
		close(self.quit)
		self.quit = nil
	}
	// if maxUncashed is set to 0, then autocash on receipt
	if cashInterval == time.Duration(0) || self.maxUncashed != nil && self.maxUncashed.Cmp(common.Big0) == 0 {
		return
	}

	ticker := time.NewTicker(cashInterval)
	self.quit = make(chan bool)
	quit := self.quit
	go func() {
	FOR:
		for {
			select {
			case <-quit:
				break FOR
			case <-ticker.C:
				self.lock.Lock()
				if self.cheque != nil && self.cheque.Amount.Cmp(self.cashed) != 0 {
					txhash, err := self.Cash()
					if err == nil {
						self.txhash = txhash
					}
				}
				self.lock.Unlock()
			}
		}
	}()
	return
}

// Receive(cheque) called to deposit latest cheque to incoming Inbox
func (self *Inbox) Receive(promise swap.Promise) (*big.Int, error) {
	ch := promise.(*Cheque)

	defer self.lock.Unlock()
	self.lock.Lock()
	var sum *big.Int
	if self.cheque == nil {
		// the sum is checked against the blockchain once a check is received
		//
		tally, err := self.session.Sent(self.beneficiary)
		if err != nil {
			return nil, fmt.Errorf("inbox: error 	calling backend to set amount: %v", err)
		}
		sum = tally
	} else {
		sum = self.cheque.Amount
	}

	amount, err := ch.Verify(self.signer, self.contract, self.beneficiary, sum)
	var uncashed *big.Int
	if err == nil {
		self.cheque = ch

		if self.maxUncashed != nil {
			uncashed = new(big.Int).Sub(ch.Amount, self.cashed)
			if self.maxUncashed.Cmp(uncashed) < 0 {
				self.Cash()
			}
		}
		glog.V(logger.Detail).Infof("[CHEQUEBOOK] received cheque of %v wei in inbox (%s, uncashed: %v)", amount, self.contract.Hex(), uncashed)
	}

	return amount, err
}

// Verify(cheque) verifies cheque for signer, contract, beneficiary, amount, valid signature
func (self *Cheque) Verify(signerKey *ecdsa.PublicKey, contract, beneficiary common.Address, sum *big.Int) (*big.Int, error) {
	glog.V(logger.Detail).Infof("[CHEQUEBOOK] verify cheque: %v - sum: %v", self, sum)
	if sum == nil {
		return nil, fmt.Errorf("invalid amount")
	}

	if self.Beneficiary != beneficiary {
		return nil, fmt.Errorf("beneficiary mismatch: %v != %v", self.Beneficiary.Hex(), beneficiary.Hex())
	}
	if self.Contract != contract {
		return nil, fmt.Errorf("contract mismatch: %v != %v", self.Contract.Hex(), contract.Hex())
	}

	amount := new(big.Int).Set(self.Amount)
	if sum != nil {
		amount.Sub(amount, sum)
		if amount.Cmp(common.Big0) <= 0 {
			return nil, fmt.Errorf("incorrect amount: %v <= 0", amount)
		}
	}

	pubKey, err := crypto.SigToPub(sigHash(self.Contract, beneficiary, self.Amount), self.Sig)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %v", err)
	}
	if !bytes.Equal(crypto.FromECDSAPub(pubKey), crypto.FromECDSAPub(signerKey)) {
		return nil, fmt.Errorf("signer mismatch: %x != %x", crypto.FromECDSAPub(pubKey), crypto.FromECDSAPub(signerKey))
	}
	return amount, nil
}

// v/r/s representation of signature
func sig2vrs(sig []byte) (v *big.Int, r, s [32]byte) {
	v = big.NewInt(int64(sig[64] + 27))
	copy(r[:], sig[:32])
	copy(s[:], sig[32:64])
	return
}

// Cash(backend) will cash the check using abi contract backend to send a transaction
// Beneficiary address should be unlocked
func (self *Cheque) Cash(session *contract.ChequebookSession) (string, error) {
	v, r, s := sig2vrs(self.Sig)
	tx, err := session.Cash(self.Beneficiary, self.Amount, v, r, s)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}
