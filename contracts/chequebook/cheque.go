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

// Package chequebook package wraps the 'chequebook' Ethereum smart contract.
//
// The functions in this package allow using chequebook for
// issuing, receiving, verifying cheques in ether; (auto)cashing cheques in ether
// as well as (auto)depositing ether to the chequebook contract.
package chequebook

//go:generate abigen --sol contract/chequebook.sol --exc contract/mortal.sol:mortal,contract/owned.sol:owned --pkg contract --out contract/chequebook.go
//go:generate go run ./gencode.go

import (
	"bytes"
	"context"
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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contracts/chequebook/contract"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/services/swap/swap"
)

// TODO(zelig): watch peer solvency and notify of bouncing cheques
// TODO(zelig): enable paying with cheque by signing off

// Some functionality requires interacting with the blockchain:
// * setting current balance on peer's chequebook
// * sending the transaction to cash the cheque
// * depositing ether to the chequebook
// * watching incoming ether

var (
	gasToCash = uint64(2000000) // gas cost of a cash transaction using chequebook
	// gasToDeploy = uint64(3000000)
)

// Backend wraps all methods required for chequebook operation.
type Backend interface {
	bind.ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	BalanceAt(ctx context.Context, address common.Address, blockNum *big.Int) (*big.Int, error)
}

// Cheque represents a payment promise to a single beneficiary.
type Cheque struct {
	Contract    common.Address // address of chequebook, needed to avoid cross-contract submission
	Beneficiary common.Address
	Amount      *big.Int // cumulative amount of all funds sent
	Sig         []byte   // signature Sign(Keccak256(contract, beneficiary, amount), prvKey)
}

func (ch *Cheque) String() string {
	return fmt.Sprintf("contract: %s, beneficiary: %s, amount: %v, signature: %x", ch.Contract.Hex(), ch.Beneficiary.Hex(), ch.Amount, ch.Sig)
}

type Params struct {
	ContractCode, ContractAbi string
}

var ContractParams = &Params{contract.ChequebookBin, contract.ChequebookABI}

// Chequebook can create and sign cheques from a single contract to multiple beneficiaries.
// It is the outgoing payment handler for peer to peer micropayments.
type Chequebook struct {
	path     string                      // path to chequebook file
	prvKey   *ecdsa.PrivateKey           // private key to sign cheque with
	lock     sync.Mutex                  //
	backend  Backend                     // blockchain API
	quit     chan bool                   // when closed causes autodeposit to stop
	owner    common.Address              // owner address (derived from pubkey)
	contract *contract.Chequebook        // abigen binding
	session  *contract.ChequebookSession // abigen binding with Tx Opts

	// persisted fields
	balance      *big.Int                    // not synced with blockchain
	contractAddr common.Address              // contract address
	sent         map[common.Address]*big.Int //tallies for beneficiaries

	txhash    string   // tx hash of last deposit tx
	threshold *big.Int // threshold that triggers autodeposit if not nil
	buffer    *big.Int // buffer to keep on top of balance for fork protection

	log log.Logger // contextual logger with the contract address embedded
}

func (cb *Chequebook) String() string {
	return fmt.Sprintf("contract: %s, owner: %s, balance: %v, signer: %x", cb.contractAddr.Hex(), cb.owner.Hex(), cb.balance, cb.prvKey.PublicKey)
}

// NewChequebook creates a new Chequebook.
func NewChequebook(path string, contractAddr common.Address, prvKey *ecdsa.PrivateKey, backend Backend) (*Chequebook, error) {
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

	cb := &Chequebook{
		prvKey:       prvKey,
		balance:      balance,
		contractAddr: contractAddr,
		sent:         sent,
		path:         path,
		backend:      backend,
		owner:        transactOpts.From,
		contract:     chbook,
		session:      session,
		log:          log.New("contract", contractAddr),
	}
	if (contractAddr != common.Address{}) {
		cb.setBalanceFromBlockChain()
		cb.log.Trace("New chequebook initialised", "owner", cb.owner, "balance", cb.balance)
	}
	return cb, nil
}

func (cb *Chequebook) setBalanceFromBlockChain() {
	balance, err := cb.backend.BalanceAt(context.TODO(), cb.contractAddr, nil)
	if err != nil {
		log.Error("Failed to retrieve chequebook balance", "err", err)
	} else {
		cb.balance.Set(balance)
	}
}

// LoadChequebook loads a chequebook from disk (file path).
func LoadChequebook(path string, prvKey *ecdsa.PrivateKey, backend Backend, checkBalance bool) (*Chequebook, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cb, _ := NewChequebook(path, common.Address{}, prvKey, backend)

	if err = json.Unmarshal(data, cb); err != nil {
		return nil, err
	}
	if checkBalance {
		cb.setBalanceFromBlockChain()
	}
	log.Trace("Loaded chequebook from disk", "path", path)

	return cb, nil
}

// chequebookFile is the JSON representation of a chequebook.
type chequebookFile struct {
	Balance  string
	Contract string
	Owner    string
	Sent     map[string]string
}

// UnmarshalJSON deserialises a chequebook.
func (cb *Chequebook) UnmarshalJSON(data []byte) error {
	var file chequebookFile
	err := json.Unmarshal(data, &file)
	if err != nil {
		return err
	}
	_, ok := cb.balance.SetString(file.Balance, 10)
	if !ok {
		return fmt.Errorf("cumulative amount sent: unable to convert string to big integer: %v", file.Balance)
	}
	cb.contractAddr = common.HexToAddress(file.Contract)
	for addr, sent := range file.Sent {
		cb.sent[common.HexToAddress(addr)], ok = new(big.Int).SetString(sent, 10)
		if !ok {
			return fmt.Errorf("beneficiary %v cumulative amount sent: unable to convert string to big integer: %v", addr, sent)
		}
	}
	return nil
}

// MarshalJSON serialises a chequebook.
func (cb *Chequebook) MarshalJSON() ([]byte, error) {
	var file = &chequebookFile{
		Balance:  cb.balance.String(),
		Contract: cb.contractAddr.Hex(),
		Owner:    cb.owner.Hex(),
		Sent:     make(map[string]string),
	}
	for addr, sent := range cb.sent {
		file.Sent[addr.Hex()] = sent.String()
	}
	return json.Marshal(file)
}

// Save persists the chequebook on disk, remembering balance, contract address and
// cumulative amount of funds sent for each beneficiary.
func (cb *Chequebook) Save() error {
	data, err := json.MarshalIndent(cb, "", " ")
	if err != nil {
		return err
	}
	cb.log.Trace("Saving chequebook to disk", cb.path)

	return ioutil.WriteFile(cb.path, data, os.ModePerm)
}

// Stop quits the autodeposit go routine to terminate
func (cb *Chequebook) Stop() {
	defer cb.lock.Unlock()
	cb.lock.Lock()
	if cb.quit != nil {
		close(cb.quit)
		cb.quit = nil
	}
}

// Issue creates a cheque signed by the chequebook owner's private key. The
// signer commits to a contract (one that they own), a beneficiary and amount.
func (cb *Chequebook) Issue(beneficiary common.Address, amount *big.Int) (*Cheque, error) {
	defer cb.lock.Unlock()
	cb.lock.Lock()

	if amount.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero (%v)", amount)
	}
	var (
		ch  *Cheque
		err error
	)
	if cb.balance.Cmp(amount) < 0 {
		err = fmt.Errorf("insufficient funds to issue cheque for amount: %v. balance: %v", amount, cb.balance)
	} else {
		var sig []byte
		sent, found := cb.sent[beneficiary]
		if !found {
			sent = new(big.Int)
			cb.sent[beneficiary] = sent
		}
		sum := new(big.Int).Set(sent)
		sum.Add(sum, amount)

		sig, err = crypto.Sign(sigHash(cb.contractAddr, beneficiary, sum), cb.prvKey)
		if err == nil {
			ch = &Cheque{
				Contract:    cb.contractAddr,
				Beneficiary: beneficiary,
				Amount:      sum,
				Sig:         sig,
			}
			sent.Set(sum)
			cb.balance.Sub(cb.balance, amount) // subtract amount from balance
		}
	}
	// auto deposit if threshold is set and balance is less then threshold
	// note this is called even if issuing cheque fails
	// so we reattempt depositing
	if cb.threshold != nil {
		if cb.balance.Cmp(cb.threshold) < 0 {
			send := new(big.Int).Sub(cb.buffer, cb.balance)
			cb.deposit(send)
		}
	}
	return ch, err
}

// Cash is a convenience method to cash any cheque.
func (cb *Chequebook) Cash(ch *Cheque) (string, error) {
	return ch.Cash(cb.session)
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
	return crypto.Keccak256(input)
}

// Balance returns the current balance of the chequebook.
func (cb *Chequebook) Balance() *big.Int {
	defer cb.lock.Unlock()
	cb.lock.Lock()
	return new(big.Int).Set(cb.balance)
}

// Owner returns the owner account of the chequebook.
func (cb *Chequebook) Owner() common.Address {
	return cb.owner
}

// Address returns the on-chain contract address of the chequebook.
func (cb *Chequebook) Address() common.Address {
	return cb.contractAddr
}

// Deposit deposits money to the chequebook account.
func (cb *Chequebook) Deposit(amount *big.Int) (string, error) {
	defer cb.lock.Unlock()
	cb.lock.Lock()
	return cb.deposit(amount)
}

// deposit deposits amount to the chequebook account.
// The caller must hold lock.
func (cb *Chequebook) deposit(amount *big.Int) (string, error) {
	// since the amount is variable here, we do not use sessions
	depositTransactor := bind.NewKeyedTransactor(cb.prvKey)
	depositTransactor.Value = amount
	chbookRaw := &contract.ChequebookRaw{Contract: cb.contract}
	tx, err := chbookRaw.Transfer(depositTransactor)
	if err != nil {
		cb.log.Warn("Failed to fund chequebook", "amount", amount, "balance", cb.balance, "target", cb.buffer, "err", err)
		return "", err
	}
	// assume that transaction is actually successful, we add the amount to balance right away
	cb.balance.Add(cb.balance, amount)
	cb.log.Trace("Deposited funds to chequebook", "amount", amount, "balance", cb.balance, "target", cb.buffer)
	return tx.Hash().Hex(), nil
}

// AutoDeposit (re)sets interval time and amount which triggers sending funds to the
// chequebook. Contract backend needs to be set if threshold is not less than buffer, then
// deposit will be triggered on every new cheque issued.
func (cb *Chequebook) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	defer cb.lock.Unlock()
	cb.lock.Lock()
	cb.threshold = threshold
	cb.buffer = buffer
	cb.autoDeposit(interval)
}

// autoDeposit starts a goroutine that periodically sends funds to the chequebook
// contract caller holds the lock the go routine terminates if Chequebook.quit is closed.
func (cb *Chequebook) autoDeposit(interval time.Duration) {
	if cb.quit != nil {
		close(cb.quit)
		cb.quit = nil
	}
	// if threshold >= balance autodeposit after every cheque issued
	if interval == time.Duration(0) || cb.threshold != nil && cb.buffer != nil && cb.threshold.Cmp(cb.buffer) >= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	cb.quit = make(chan bool)
	quit := cb.quit

	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				cb.lock.Lock()
				if cb.balance.Cmp(cb.buffer) < 0 {
					amount := new(big.Int).Sub(cb.buffer, cb.balance)
					txhash, err := cb.deposit(amount)
					if err == nil {
						cb.txhash = txhash
					}
				}
				cb.lock.Unlock()
			}
		}
	}()
}

// Outbox can issue cheques from a single contract to a single beneficiary.
type Outbox struct {
	chequeBook  *Chequebook
	beneficiary common.Address
}

// NewOutbox creates an outbox.
func NewOutbox(cb *Chequebook, beneficiary common.Address) *Outbox {
	return &Outbox{cb, beneficiary}
}

// Issue creates cheque.
func (o *Outbox) Issue(amount *big.Int) (swap.Promise, error) {
	return o.chequeBook.Issue(o.beneficiary, amount)
}

// AutoDeposit enables auto-deposits on the underlying chequebook.
func (o *Outbox) AutoDeposit(interval time.Duration, threshold, buffer *big.Int) {
	o.chequeBook.AutoDeposit(interval, threshold, buffer)
}

// Stop helps satisfy the swap.OutPayment interface.
func (o *Outbox) Stop() {}

// String implements fmt.Stringer.
func (o *Outbox) String() string {
	return fmt.Sprintf("chequebook: %v, beneficiary: %s, balance: %v", o.chequeBook.Address().Hex(), o.beneficiary.Hex(), o.chequeBook.Balance())
}

// Inbox can deposit, verify and cash cheques from a single contract to a single
// beneficiary. It is the incoming payment handler for peer to peer micropayments.
type Inbox struct {
	lock        sync.Mutex
	contract    common.Address              // peer's chequebook contract
	beneficiary common.Address              // local peer's receiving address
	sender      common.Address              // local peer's address to send cashing tx from
	signer      *ecdsa.PublicKey            // peer's public key
	txhash      string                      // tx hash of last cashing tx
	session     *contract.ChequebookSession // abi contract backend with tx opts
	quit        chan bool                   // when closed causes autocash to stop
	maxUncashed *big.Int                    // threshold that triggers autocashing
	cashed      *big.Int                    // cumulative amount cashed
	cheque      *Cheque                     // last cheque, nil if none yet received
	log         log.Logger                  // contextual logger with the contract address embedded
}

// NewInbox creates an Inbox. An Inboxes is not persisted, the cumulative sum is updated
// from blockchain when first cheque is received.
func NewInbox(prvKey *ecdsa.PrivateKey, contractAddr, beneficiary common.Address, signer *ecdsa.PublicKey, abigen bind.ContractBackend) (*Inbox, error) {
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

	inbox := &Inbox{
		contract:    contractAddr,
		beneficiary: beneficiary,
		sender:      sender,
		signer:      signer,
		session:     session,
		cashed:      new(big.Int).Set(common.Big0),
		log:         log.New("contract", contractAddr),
	}
	inbox.log.Trace("New chequebook inbox initialized", "beneficiary", inbox.beneficiary, "signer", hexutil.Bytes(crypto.FromECDSAPub(signer)))
	return inbox, nil
}

func (i *Inbox) String() string {
	return fmt.Sprintf("chequebook: %v, beneficiary: %s, balance: %v", i.contract.Hex(), i.beneficiary.Hex(), i.cheque.Amount)
}

// Stop quits the autocash goroutine.
func (i *Inbox) Stop() {
	defer i.lock.Unlock()
	i.lock.Lock()
	if i.quit != nil {
		close(i.quit)
		i.quit = nil
	}
}

// Cash attempts to cash the current cheque.
func (i *Inbox) Cash() (string, error) {
	if i.cheque == nil {
		return "", nil
	}
	txhash, err := i.cheque.Cash(i.session)
	i.log.Trace("Cashing in chequebook cheque", "amount", i.cheque.Amount, "beneficiary", i.beneficiary)
	i.cashed = i.cheque.Amount

	return txhash, err
}

// AutoCash (re)sets maximum time and amount which triggers cashing of the last uncashed
// cheque if maxUncashed is set to 0, then autocash on receipt.
func (i *Inbox) AutoCash(cashInterval time.Duration, maxUncashed *big.Int) {
	defer i.lock.Unlock()
	i.lock.Lock()
	i.maxUncashed = maxUncashed
	i.autoCash(cashInterval)
}

// autoCash starts a loop that periodically clears the last cheque
// if the peer is trusted. Clearing period could be 24h or a week.
// The caller must hold lock.
func (i *Inbox) autoCash(cashInterval time.Duration) {
	if i.quit != nil {
		close(i.quit)
		i.quit = nil
	}
	// if maxUncashed is set to 0, then autocash on receipt
	if cashInterval == time.Duration(0) || i.maxUncashed != nil && i.maxUncashed.Sign() == 0 {
		return
	}

	ticker := time.NewTicker(cashInterval)
	i.quit = make(chan bool)
	quit := i.quit

	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				i.lock.Lock()
				if i.cheque != nil && i.cheque.Amount.Cmp(i.cashed) != 0 {
					txhash, err := i.Cash()
					if err == nil {
						i.txhash = txhash
					}
				}
				i.lock.Unlock()
			}
		}
	}()
}

// Receive is called to deposit the latest cheque to the incoming Inbox.
// The given promise must be a *Cheque.
func (i *Inbox) Receive(promise swap.Promise) (*big.Int, error) {
	ch := promise.(*Cheque)

	defer i.lock.Unlock()
	i.lock.Lock()

	var sum *big.Int
	if i.cheque == nil {
		// the sum is checked against the blockchain once a cheque is received
		tally, err := i.session.Sent(i.beneficiary)
		if err != nil {
			return nil, fmt.Errorf("inbox: error calling backend to set amount: %v", err)
		}
		sum = tally
	} else {
		sum = i.cheque.Amount
	}

	amount, err := ch.Verify(i.signer, i.contract, i.beneficiary, sum)
	var uncashed *big.Int
	if err == nil {
		i.cheque = ch

		if i.maxUncashed != nil {
			uncashed = new(big.Int).Sub(ch.Amount, i.cashed)
			if i.maxUncashed.Cmp(uncashed) < 0 {
				i.Cash()
			}
		}
		i.log.Trace("Received cheque in chequebook inbox", "amount", amount, "uncashed", uncashed)
	}

	return amount, err
}

// Verify verifies cheque for signer, contract, beneficiary, amount, valid signature.
func (ch *Cheque) Verify(signerKey *ecdsa.PublicKey, contract, beneficiary common.Address, sum *big.Int) (*big.Int, error) {
	log.Trace("Verifying chequebook cheque", "cheque", ch, "sum", sum)
	if sum == nil {
		return nil, fmt.Errorf("invalid amount")
	}

	if ch.Beneficiary != beneficiary {
		return nil, fmt.Errorf("beneficiary mismatch: %v != %v", ch.Beneficiary.Hex(), beneficiary.Hex())
	}
	if ch.Contract != contract {
		return nil, fmt.Errorf("contract mismatch: %v != %v", ch.Contract.Hex(), contract.Hex())
	}

	amount := new(big.Int).Set(ch.Amount)
	if sum != nil {
		amount.Sub(amount, sum)
		if amount.Sign() <= 0 {
			return nil, fmt.Errorf("incorrect amount: %v <= 0", amount)
		}
	}

	pubKey, err := crypto.SigToPub(sigHash(ch.Contract, beneficiary, ch.Amount), ch.Sig)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %v", err)
	}
	if !bytes.Equal(crypto.FromECDSAPub(pubKey), crypto.FromECDSAPub(signerKey)) {
		return nil, fmt.Errorf("signer mismatch: %x != %x", crypto.FromECDSAPub(pubKey), crypto.FromECDSAPub(signerKey))
	}
	return amount, nil
}

// v/r/s representation of signature
func sig2vrs(sig []byte) (v byte, r, s [32]byte) {
	v = sig[64] + 27
	copy(r[:], sig[:32])
	copy(s[:], sig[32:64])
	return
}

// Cash cashes the cheque by sending an Ethereum transaction.
func (ch *Cheque) Cash(session *contract.ChequebookSession) (string, error) {
	v, r, s := sig2vrs(ch.Sig)
	tx, err := session.Cash(ch.Beneficiary, ch.Amount, v, r, s)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

// ValidateCode checks that the on-chain code at address matches the expected chequebook
// contract code. This is used to detect suicided chequebooks.
func ValidateCode(ctx context.Context, b Backend, address common.Address) (bool, error) {
	code, err := b.CodeAt(ctx, address, nil)
	if err != nil {
		return false, err
	}
	return bytes.Equal(code, common.FromHex(contract.ContractDeployedCode)), nil
}
