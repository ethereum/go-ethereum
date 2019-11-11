// Copyright 2019 The go-ethereum Authors
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

package payment

import (
	"context"
	"errors"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/accountbook"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var errInvalidOpt = errors.New("invalid operation")

var (
	minimalDeposit           = big.NewInt(1e6) // The minimal amount for single deposit operation, 1e6 gWei.
	minimalDepositThreshold  = big.NewInt(2e5) // The minimal amount for triggering a new deposit, 2e5 gWei
	minimalCashThreshold     = big.NewInt(1e6) // The minimal amount for triggering a cash operation, 1e6 gWei
	minimalChallengThreshold = big.NewInt(2e5) // The minimal amount for triggering a challenge, 2e5 gWei
)

// DefaultPaymentChannelDraweeConfig is the default payment channel
// config for drawee.
var DefaultPaymentChannelDraweeConfig = &PaymentChannelConfig{
	Role:     PaymentDrawee,
	AutoCash: true,

	// The transaction fee of cash call is around 4e14 wei
	// The gas cost is around 60,000, we use 10GW as the price.
	// So a reasonable minimal call amount is 1e16 wei as well
	// as 1e-2 ether.
	AutoCashThreshold:  big.NewInt(int64(1e7)), // 1e7 gWei as well as 1e-2 ether
	ChallengeThreshold: big.NewInt(int64(1e6)), // 1e6 gWei as well as 1e-3 ether
}

// DefaultPaymentChannelDrawerConfig is the default payment channel
// config for drawer.
var DefaultPaymentChannelDrawerConfig = &PaymentChannelConfig{
	Role:        PaymentDrawer,
	AutoDeposit: true,
	AutoClaim:   true,

	// The transaction fee of deposit call is around 4e14 wei
	// The gas cost is around 40,000, we use 10GW as the price.
	// So a reasonable minimal deposit amount is 1e16 wei as well
	// as 1e-2 ether.
	DepositAmount:        big.NewInt(int64(1e7)), // 1e7 gWei as well as 1e-2 ether
	AutoDepositThreshold: big.NewInt(int64(1e6)), // 1e6 gWei as well as 1e-3 ether
}

// PaymentRole is the role of user in payment channel.
type PaymentRole int

const (
	PaymentDrawer PaymentRole = iota
	PaymentDrawee
)

// PaymentChannelConfig defines all user-selectable options for both
// drawer and drawee.
type PaymentChannelConfig struct {
	// Role is the role of the user in the payment channel, either the
	// payer or the payee.
	Role PaymentRole

	// Drawer relative options

	// TrustedContract is a list of contract code hash which light client
	// can trust for usage. Light client users can configure or extend it
	// by themselves, but as default the in-built contract code hash is
	// included(for light clients).
	TrustedContracts []common.Hash

	// DepositAmount is the amount deposited by each drawer for the deposit
	// operation. The unit is gWei. This option is only for drawer.
	DepositAmount *big.Int

	// AutoDeposit is the indicator whether to perform automatic balance
	// recharge. This option is only for drawer.
	AutoDeposit bool

	// AutoDepositThreshold is the threshold for the drawer to perform auto deposit.
	// When balance is below the threshold, auto deposit is triggered. This option is
	// only for drawer.
	AutoDepositThreshold *big.Int

	// AutoClaim is the indicator whether to perform automatic claim.
	AutoClaim bool

	// Drawee relative options

	// AutoCash is an indicator that whether drawee to perform cheque cashing
	// automatically. This option is only for drawee.
	AutoCash bool

	// AutoCashThreshold is the threshold for the drawee to perform auto cashing.
	// When accumulated received money from single drawer exceeds the threshold,
	// auto cashing is triggered. This option is only for drawee.
	AutoCashThreshold *big.Int

	// ChallengeThreshold is the threshold for drawee to initiate the challenge to
	// the drawer's withdraw operation.
	//
	// If the drawer tries to withdraw the deposit that has been spent from the
	// contract, the drawee can initiate the challenge. But this needs to be done
	// through an on-chain transaction. So the drawer can choose to set the threshold
	// and initiate the challenge when the extracted amount exceeds the value.
	ChallengeThreshold *big.Int
}

// sanitize checks the provided user configurations and changes anything that's
// unreasonable or unworkable.
func (config *PaymentChannelConfig) sanitize() *PaymentChannelConfig {
	conf := *config
	if conf.Role != PaymentDrawer && conf.Role != PaymentDrawee {
		return nil
	}
	if conf.Role == PaymentDrawer {
		// If auto deposit is disabled, return directly.
		if !conf.AutoDeposit {
			return &conf
		}
		if conf.DepositAmount == nil || conf.DepositAmount.Cmp(minimalDeposit) < 0 {
			log.Warn("Sanitizing invalid deposit amount", "provided(gWei)", conf.DepositAmount, "updated(gWei)", minimalDeposit)
			conf.DepositAmount = minimalDeposit
		}
		if conf.AutoDepositThreshold == nil || conf.AutoDepositThreshold.Cmp(minimalDepositThreshold) < 0 {
			log.Warn("Sanitizing invalid deposit threshold", "provided(gWei)", conf.AutoDepositThreshold, "updated(gWei)", minimalDepositThreshold)
			conf.AutoDepositThreshold = minimalDepositThreshold
		}
	} else {
		if conf.AutoCash && (conf.AutoCashThreshold == nil || conf.AutoCashThreshold.Cmp(minimalCashThreshold) < 0) {
			log.Warn("Sanitizing invalid cash threshold", "provided(gWei)", conf.AutoCashThreshold, "updated(gWei)", minimalCashThreshold)
			conf.AutoCashThreshold = minimalCashThreshold
		}
		if conf.ChallengeThreshold == nil || conf.ChallengeThreshold.Cmp(minimalChallengThreshold) < 0 {
			log.Warn("Sanitizing invalid challenge threshold", "provided(gWei)", conf.ChallengeThreshold, "updated(gWei)", minimalChallengThreshold)
			conf.ChallengeThreshold = minimalChallengThreshold
		}
	}
	return &conf
}

// Peer defines all necessary method as the drawer or drawee.
type Peer interface {
	// SendPayment sends the given cheque to the peer via network.
	SendPayment(cheque *accountbook.Cheque) error

	// AddBalance notifies upper-level system we have received
	// the payment from the peer with specified amount.
	AddBalance(amount *big.Int) error
}

// CurrentHeader retrieves the current header from the local chain.
type ChainReader interface {
	CurrentHeader() *types.Header
}

type PaymentChannel struct {
	config      *PaymentChannelConfig
	chainReader ChainReader
	chanAddr    common.Address
	peer        Peer                      // The peer handler of counterparty
	drawer      *accountbook.ChequeDrawer // Nil if payment is opened by drawee
	drawee      *accountbook.ChequeDrawee // Nil if payment is opened by drawer

	depositCh chan struct{}
	cashCh    chan common.Address
	closeCh   chan struct{}
	wg        sync.WaitGroup
}

func NewPaymentChannel(config *PaymentChannelConfig, chainReader ChainReader, chanAddr common.Address, drawer *accountbook.ChequeDrawer, drawee *accountbook.ChequeDrawee, peer Peer) (*PaymentChannel, error) {
	// Sanitize the config to ensure all options are valid
	checked := config.sanitize()
	if checked == nil {
		return nil, errors.New("invalid config")
	}
	payment := &PaymentChannel{
		config:      checked,
		chainReader: chainReader,
		chanAddr:    chanAddr,
		peer:        peer,
		drawer:      drawer,
		drawee:      drawee,
		depositCh:   make(chan struct{}),
		cashCh:      make(chan common.Address),
		closeCh:     make(chan struct{}),
	}
	if config.Role == PaymentDrawer {
		if config.AutoDeposit {
			payment.wg.Add(1)
			go payment.autoDeposit()
		}
		if config.AutoClaim {
			payment.wg.Add(1)
			go payment.autoClaim()
		}
	} else {
		if config.AutoCash {
			payment.wg.Add(1)
			go payment.autoCash()
		}
		payment.wg.Add(1)
		go payment.listenWithdraw()
	}
	return payment, nil
}

// Pay initiates a payment to the designated payee with specified
// payemnt amount and also trigger a deposit operation if auto deposit
// is set and threshold is met.
func (c *PaymentChannel) Pay(amount *big.Int) error {
	if c.config.Role != PaymentDrawer {
		return errInvalidOpt
	}
	cheque, unspent, err := c.drawer.IssueCheque(amount)
	if err != nil {
		if c.config.AutoDeposit && err == accountbook.ErrNotEnoughDeposit {
			select {
			case c.depositCh <- struct{}{}:
			case <-c.closeCh:
			}
		}
		return err
	}
	if c.config.AutoDeposit && unspent.Cmp(new(big.Int).Mul(c.config.AutoDepositThreshold, big.NewInt(params.GWei))) <= 0 {
		select {
		case c.depositCh <- struct{}{}:
		case <-c.closeCh:
		}
	}
	log.Info("Issued payment", "amount", amount, "channel", c.chanAddr)
	return c.peer.SendPayment(cheque)
}

// Receive receives a payment from the payer and returns any error
// for payment processing and proving.
func (c *PaymentChannel) Receive(msg io.Reader) error {
	if c.config.Role != PaymentDrawee {
		return errInvalidOpt
	}
	var cheque accountbook.Cheque
	if err := rlp.Decode(msg, &cheque); err != nil {
		return err
	}
	amount, unpaid, err := c.drawee.AddCheque(&cheque)
	if err != nil {
		return err
	}
	if c.config.AutoCash && unpaid.Cmp(new(big.Int).Mul(c.config.AutoCashThreshold, big.NewInt(params.GWei))) >= 0 {
		select {
		case c.cashCh <- cheque.Drawer:
		case <-c.closeCh:
		}
	}
	return c.peer.AddBalance(amount)
}

// Amend amends the local cheque db of drawer with externally provided cheque
// signed by drawer itself.
func (c *PaymentChannel) Amend(msg io.Reader) error {
	if c.config.Role != PaymentDrawer {
		return errInvalidOpt
	}
	var serr accountbook.StaleChequeError
	if err := rlp.Decode(msg, &serr); err != nil {
		return err
	}
	return c.drawer.Amend(serr.Evidence)
}

// Close exits the payment and opens the reqeust to withdraw all funds.
func (c *PaymentChannel) Close() error {
	if c.config.Role != PaymentDrawer {
		return errInvalidOpt
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancelFn()

	if err := c.drawer.Withdraw(ctx); err != nil {
		log.Info("Failed to open withdraw request", "error", err)
		return err
	}
	return nil
}

func (c *PaymentChannel) autoCash() {
	log.Info("Enable auto cash", "channel", c.chanAddr, "threshold", c.config.AutoCashThreshold)
	defer c.wg.Done()

	var (
		done chan struct{} // Non-nil if cash routine is active.
		cash = func(drawer common.Address, done chan struct{}) {
			defer func() { done <- struct{}{} }()

			ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute*5)
			defer cancelFn()

			if err := c.drawee.Cash(ctx, drawer, true); err != nil {
				log.Info("Failed to cash payment", "drawer", drawer, "error", err)
			} else {
				log.Info("Succeed to cash payment", "drawer", drawer)
			}
		}
	)
	for {
		select {
		case addr := <-c.cashCh:
			if done == nil {
				done = make(chan struct{})
				go cash(addr, done)
			}
		case <-done:
			done = nil
		case <-c.closeCh:
			return
		}
	}
}

func (c *PaymentChannel) autoDeposit() {
	log.Info("Enable auto deposit", "channel", c.chanAddr, "amount", c.config.DepositAmount)
	defer c.wg.Done()

	var (
		done    chan struct{} // Non-nil if deposit routine is active.
		deposit = func(done chan struct{}) {
			defer func() { done <- struct{}{} }()

			ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute*5)
			defer cancelFn()

			status, err := c.drawer.Deposit(ctx, new(big.Int).Mul(c.config.DepositAmount, big.NewInt(params.GWei)))
			if err != nil || !status {
				log.Info("Failed to deposit", "channel", c.chanAddr, "amount(gWei)", c.config.DepositAmount, "error", err)
			} else {
				log.Info("Succeed to deposit", "channel", c.chanAddr, "amount(gWei)", c.config.DepositAmount)
			}
		}
	)
	for {
		select {
		case <-c.depositCh:
			if done == nil {
				done = make(chan struct{})
				go deposit(done)
			}
		case <-done:
			done = nil
		case <-c.closeCh:
			return
		}
	}
}

func (c *PaymentChannel) autoClaim() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	var (
		done  chan struct{} // Non-nil if deposit routine is active.
		claim = func(done chan struct{}) {
			defer func() { done <- struct{}{} }()
			defer c.drawer.ResetWithdrawlRecord()

			ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute*5)
			defer cancelFn()

			if status, err := c.drawer.Claim(ctx); err != nil || !status {
				log.Info("Failed to claim", "channel", c.chanAddr, "error", err)
			} else {
				log.Info("Succeed to claim", "channel", c.chanAddr)
			}
		}
	)
	for {
		select {
		case <-ticker.C:
			if done != nil {
				continue
			}
			createdAt := c.drawer.WithdrawalRecord()
			if createdAt != 0 {
				local := c.chainReader.CurrentHeader()
				if local.Number.Uint64() > createdAt && local.Number.Uint64()-createdAt > accountbook.ChallengeTimeWindow {
					done = make(chan struct{})
					go claim(done)
				}
			}
		case <-done:
			done = nil
		case <-c.closeCh:
			return
		}
	}
}

func (c *PaymentChannel) listenWithdraw() {
	defer c.wg.Done()

	sub, channel, err := c.drawee.ListenWithdraw()
	if err != nil {
		log.Info("Failed to subscribe withdraw event", "error", err)
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case event := <-channel:
			amount := event.Amount
			if amount == nil || amount.Cmp(new(big.Int).Mul(c.config.ChallengeThreshold, big.NewInt(params.GWei))) < 0 {
				continue
			}
			_, err := c.drawee.Unspent(event.Addr)
			if err != nil && err != accountbook.ErrNotEnoughDeposit {
				continue
			}
			// Drawer tries to withdraw spent money, challenge him.
			go func() {
				if err := c.drawee.Cash(context.Background(), event.Addr, false); err != nil {
					log.Info("Failed to challenge", "drawer", event.Addr, "error", err)
				} else {
					log.Info("Succeed to challenge", "drawer", event.Addr, "error", err)
				}
			}()
		case <-c.closeCh:
			return
		}
	}
}

func (c *PaymentChannel) exit() {
	close(c.closeCh)
	c.wg.Wait()
	return
}

type PaymentChannelManager struct {
	config       *PaymentChannelConfig
	chainReader  ChainReader
	localAddr    common.Address
	txSigner     *bind.TransactOpts
	chequeSigner func(data []byte) ([]byte, error)
	db           ethdb.Database
	lock         sync.RWMutex

	// payments are all established channels. For payment drawer, the key of payment
	// map is channel contract address, otherwise the key refers to drawer's address.
	payments map[common.Address]*PaymentChannel
	drawee   *accountbook.ChequeDrawee // Nil if manager is opened by drawer

	// Backends used to interact with the underlying payment contract
	cBackend bind.ContractBackend
	dBackend bind.DeployBackend
}

// NewPaymentChannel initializes a one-to-one payment channel for both
// drawer and drawee.
func NewPaymentChannelManager(config *PaymentChannelConfig, chainReader ChainReader, txSigner *bind.TransactOpts, chequeSigner func(digestHash []byte) ([]byte, error), localAddr common.Address, cBackend bind.ContractBackend, dBackend bind.DeployBackend, db ethdb.Database) (*PaymentChannelManager, error) {
	c := &PaymentChannelManager{
		config:       config,
		chainReader:  chainReader,
		localAddr:    localAddr,
		txSigner:     txSigner,
		chequeSigner: chequeSigner,
		db:           db,
		cBackend:     cBackend,
		dBackend:     dBackend,
		payments:     make(map[common.Address]*PaymentChannel),
	}
	// Drawer has to initialize channel here if contract
	// hasn't been deployed yet.
	if c.config.Role == PaymentDrawee {
		drawee, err := accountbook.NewChequeDrawee(c.localAddr, txSigner, cBackend, dBackend, db)
		if err != nil {
			return nil, err
		}
		c.drawee = drawee
	}
	return c, nil
}

// OpenChannel establishes a new payment channel for new customer or new vendor.
// If we are payment drawer, the addr refers to the payment channel contract addr,
// otherwise, the addr refers to drawer's address.
func (c *PaymentChannelManager) OpenChannel(addr common.Address, peer Peer) (Payment, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Filter all duplicated channels.
	if _, exist := c.payments[addr]; exist {
		return nil, errors.New("duplicated payment channel")
	}
	var err error
	var channel *PaymentChannel
	if c.config.Role == PaymentDrawer {
		// We are payment drawer, establish a outgoing channel with
		// specified contract address and counterparty peer.
		drawer, err := accountbook.NewChequeDrawer(c.txSigner, c.chequeSigner, c.localAddr, addr, c.cBackend, c.dBackend, c.db)
		if err != nil {
			return nil, err
		}
		channel, err = NewPaymentChannel(c.config, c.chainReader, addr, drawer, nil, peer)
		if err != nil {
			return nil, err
		}
		c.payments[addr] = channel
	} else {
		// We are payment drawee, establish a incoming channel with
		// specified counterparty address and peer.
		channel, err = NewPaymentChannel(c.config, c.chainReader, c.drawee.ContractAddr(), nil, c.drawee, peer)
		if err != nil {
			return nil, err
		}
		c.payments[addr] = channel
	}
	return channel, nil
}

// CloseChannel closes a channel with given address. If we are payment drawer,
// the addr refers to the payment channel contract addr, otherwise, the addr
// refers to drawer's address.
func (c *PaymentChannelManager) CloseChannel(addr common.Address) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if payment, exist := c.payments[addr]; !exist {
		return errors.New("channel doesn't exist")
	} else {
		payment.exit()
		delete(c.payments, addr)
	}
	return nil
}

// VerifyChannel ensures the code of payment channel is trusted.
func (c *PaymentChannelManager) VerifyChannel(chanAddr common.Address) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.config.Role == PaymentDrawee {
		return nil
	}
	if len(c.config.TrustedContracts) == 0 {
		return nil
	}
	payment, exist := c.payments[chanAddr]
	if !exist {
		return errors.New("channel doesn't exist")
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancelFn()
	hash, err := payment.drawer.CodeHash(ctx)
	if err != nil {
		return err
	}
	for _, h := range c.config.TrustedContracts {
		if h == hash {
			return nil
		}
	}
	return errors.New("untrusted contract")
}

// ChannelAddress returns all established channel addresses.
func (c *PaymentChannelManager) ChannelAddresses() []common.Address {
	if c.config.Role == PaymentDrawer {
		var addresses []common.Address
		c.lock.RLock()
		defer c.lock.RUnlock()
		for addr := range c.payments {
			addresses = append(addresses, addr)
		}
		return addresses
	}
	return []common.Address{c.drawee.ContractAddr()}
}

// ChannelInfo includes all basic information about the specified channel.
type ChannelInfo struct {
	Received []*accountbook.Cheque
	Payed    *big.Int
}

// ChannelInfo returns all basic information about the channel.
func (c *PaymentChannelManager) ChannelInfo(chanAddr common.Address) ChannelInfo {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var info ChannelInfo
	if c.config.Role == PaymentDrawer {
		payment, exist := c.payments[chanAddr]
		if !exist {
			return info
		}
		info.Payed = payment.drawer.Payed()
	} else {
		info.Received = c.drawee.ListCheques()
	}
	return info
}
