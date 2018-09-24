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
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network/stream"
	"github.com/ethereum/go-ethereum/swarm/state"
)

const (
	defaultMaxMsgSize = 1024 * 1024
	swapProtocolName  = "swap"
	swapVersion       = 1
)

var (
	payAt  = big.NewInt(-4096 * 10000) // threshold that triggers payment {request} (bytes)
	dropAt = big.NewInt(-4096 * 12000) // threshold that triggers disconnect (bytes)

	ErrInsufficientFunds = errors.New("Insufficient funds")
	ErrNotAccountedMsg   = errors.New("Message does not need accounting")
)

const (
	chequebookDeployRetries = 5
	chequebookDeployDelay   = 1 * time.Second // delay between retries
)

// SwAP Swarm Accounting Protocol
// a peer to peer micropayment system
// A node maintains an individual balance with every peer
// Only messages which have a price will be accounted for
type Swap struct {
	priceOracle   PriceOracle
	chequeManager *ChequeManager
	stateStore    state.Store
	lock          sync.RWMutex
	peers         map[discover.NodeID]*SwapPeer
}

//Protocols which want to send and handle priced messages will need to use
//this peer instead of protocols.Peer, which is embedded
type SwapPeer struct {
	*protocols.Peer
	lock        sync.RWMutex
	swapAccount *Swap
	handlerFunc func(context.Context, interface{}) error
	balance     *big.Int
	storeID     string
}

type PriceOracle interface {
	IsAccountedMsg(event *p2p.PeerEvent) bool
	GetPriceForMsg(event *p2p.PeerEvent) *PriceTag
}

//This defines if a price will be debited or credited to an account
type EntryDirection bool

const (
	ChargeSender   EntryDirection = true
	ChargeReceiver EntryDirection = false
)

type PriceTag struct {
	Price     *big.Int
	SizeBased bool
	Direction EntryDirection
}

type DefaultPriceOracle struct {
	priceMatrix map[string]map[int]*PriceTag
}

func (dpo *DefaultPriceOracle) LoadPriceMatrix() {
	priceMatrix := dpo.loadDefaultPriceMatrix()
}

func (dpo *DefaultPriceOracle) loadDefaultPriceMatrix() map[string]map[int]*PriceTag {
	priceMatrix := make(map[string]map[int]*big.Int)

	streamProtocol := stream.Spec.Name

	priceMatrix[streamProtocol] = make(map[int]*PriceTag)

	retrieveRequestMsgIndex := dpo.getIndexForMsgType(stream.RetrieveRequestMsg, stream.Spec.Messages)
	deliveryMsgIndex := dpo.getIndexForMsgType(stream.ChunkDeliveryMsg, stream.Spec.Messages)

	priceMatrix[streamProtocol][retrieveRequestMsgIndex] = &PriceTag{
		Price:     big.NewInt(10),
		SizeBased: false,
		Direction: ChargeSender,
	}
	priceMatrix[streamProtocol][deliveryMsgIndex] = &PriceTag{
		Price:     big.NewInt(100),
		SizeBased: true,
		Direction: ChargeReceiver,
	}

	return priceMatrix

}

func (dpo *DefaultPriceOracle) getIndexForMsgType(iface interface{}, types []interface{}) {
	for i, v := range types {
		if iface == v {
			return i
		}
	}
	return -1
}

func (dpo *DefaultPriceOracle) IsAccountedMsg(event *p2p.PeerEvent) bool {
	if protoPriceMap, ok := dpo.priceMatrix[event.Protocol]; !ok {
		return false
	}
	if msgCode, ok := protoPriceMap[event.MsgCode]; !ok {
		return false
	}
	return true
}

func (dpo *DefaultPriceOracle) GetPriceForMsg(event *p2p.PeerEvent) (*big.Int, EntryDirection) {
	if dpo.IsAccountedMsg(event) {
		priceTag := dpo.priceMatrix[event.Protocol][event.MsgCode]
		price := priceTag.Price
		if priceTag.SizeBased {
			price = price * event.MsgSize
		}
		return price, priceTag.Direction
	}
	return nil, false
}

//A message which needs accounting needs to implement this interface
type PricedMsg interface {
	Price() *big.Int
}

func (s *Swap) RegisterForEvents(srv *p2p.Server) {
	go func() {
		events := make(chan *p2p.PeerEvent)
		sub := server.SubscribeEvents(events)
		defer sub.Unsubscribe()

		for {
			select {
			case event := <-events:
				go handleMsgEvent(event)
			case <-sub.Err():
				return
			}
		}
	}()
}

func (s *Swap) handleMsgEvent(event *p2p.PeerEvent) {
	if !s.priceOracle.IsAccountedMsg(event) {
		return
	}
	s.AccountMsgForPeer(event)
}

func (s *Swap) AccountMsgForPeer(event *p2p.PeerEvent) {
	price, direction := s.priceOracle.GetPriceForMsg(event)
	if price == nil {
		//TODO what to do in this case? Should not happen
		log.Crit("Price is nil; this should have been accounted for but somehow it failed")
	}

	if direction == ChargeSender {
		if event.Type == p2p.PeerEventTypeMsgSend {
			s.chargeSender(event, price)
		} else if event.Type == p2p.PeerEventTypeMsgRecv {
			s.chargeReceiver(event, price)
		}
	} else {
		if event.Type == p2p.PeerEventTypeMsgRecv {
			s.chargeReceiver(event, price)
		} else if event.Type == p2p.PeerEventTypeMsgSend {
			s.chargeSender(event, price)
		}
	}

}

//Handler for received messages
func (sp *SwapPeer) RunAccountedProtocol(protocolHandler func(ctx context.Context, msg interface{}) error) error {
	//the `peer.Run` function is a loop, so in order to pre-/post-process a message with accounting,
	//we need to save the actual handler
	sp.handlerFunc = protocolHandler

	//then run the handler loop function
	return sp.Run(sp.handle)
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
func (sp *SwapPeer) handle(ctx context.Context, msg interface{}) error {
	var err error
	var price *big.Int

	//the message is one which needs accounting...
	//only account if swapAccount != nil (== swap is disabled)
	if _, ok := msg.(PricedMsg); ok && sp.swapAccount != nil {
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
	//only account if swapAccount != nil (== swap is disabled)
	if _, ok := msg.(PricedMsg); ok && sp.swapAccount != nil {
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

	if accounted, ok := msg.(PricedMsg); ok {
		price := accounted.Price()
		//local node is being credited (in its favor), so check upper limit
		if direction == CreditEntry {
			//TODO: is there a check needed here?
			//It should actually have been done on the client side, the debitor!
			//creditor could theoretically go over payAt, but if well done,
			//should have been checked on the client side so this shouldn't happen?
			checkBalance := &big.Int{}
			checkBalance.Add(sp.balance, price)
			//(checkBalance *Int) CmpAbs(payAt)
			// -1 if |checkBalance|  <  |payAt|
			//  0 if |checkBalance|  == |payAt|
			// +1 if |checkBalance|  >  |payAt|
			if checkBalance.CmpAbs(payAt) == 1 {
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
			checkBalance := &big.Int{}
			checkBalance.Sub(sp.balance, price.Abs(price))
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
	if _, ok := msg.(PricedMsg); ok {
		sp.lock.Lock()
		defer sp.lock.Unlock()
		//local node is being credited (in its favor), so its balance increases
		if direction == CreditEntry {
			//NOTE: do we need to check for sufficient funds again?
			//operations are not atomic/transactional, so balance may have changed in the meanwhile!
			sp.balance.Add(sp.balance, price)
			//local node is being debited (in favor of remote peer), so its balance decreases
		} else if direction == DebitEntry {
			sp.balance.Sub(sp.balance, price)
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
				log.Warn("Payment threshold exceeded, but error sending cheque!", "err", err)
			}
		}
		if sp.balance.Cmp(dropAt) == -1 {
			sp.Drop(ErrInsufficientFunds)
		}
		log.Debug(fmt.Sprintf("balance for peer %s: %s", sp.ID(), sp.balance.String()))
	}
}

//Issue a cheque for the remote peer. Happens if we are indebted with the peer
//and crossed the payment threshold
func (sp *SwapPeer) issueCheque(ctx context.Context) error {
	amount := &big.Int{}
	cheque := sp.swapAccount.chequeManager.CreateCheque(sp.ID(), amount.Abs(payAt))
	msg := IssueChequeMsg{
		Cheque: cheque,
	}
	//TODO: This should now be via the actual SwapProtocol
	return sp.Send(ctx, msg)
}

//Create a new swap accounted peer
func NewSwapPeer(peer *protocols.Peer, swap *Swap) *SwapPeer {
	sp := &SwapPeer{
		Peer:        peer,
		swapAccount: swap,
		storeID:     peer.String()[:24] + "-swap",
	}
	//swap is not enabled
	if swap != nil {
		//check if there is one already in the stateStore and load it
		balance := &big.Int{}
		swap.stateStore.Get(peer.String()[:24]+"-swap", &balance)
		sp.balance = balance
		swap.lock.Lock()
		defer swap.lock.Unlock()
		swap.peers[peer.ID()] = sp
	}
	return sp
}

// New - swap constructor
func New(stateStore state.Store, srv *p2p.Server) (swap *Swap, err error) {

	swap = &Swap{
		chequeManager: NewChequeManager(stateStore),
		stateStore:    stateStore,
		peers:         make(map[discover.NodeID]*SwapPeer),
		priceOracle:   &DefaultPriceOracle{},
	}

	swap.RegisterForEvents(srv)

	return
}
