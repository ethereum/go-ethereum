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

	ErrNotAccountedMsg   = errors.New("Message does not need accounting")
	ErrInsufficientFunds = errors.New("Insufficient funds")
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
	peers         map[discover.NodeID]*big.Int
	p2pServer     *p2p.Server
	protocol      *Protocol
}

//PriceOracle is responsible to maintain the price matrix for accounted messages,
//to get its price and to evaluate if a message is accountable or not
type PriceOracle interface {
	IsAccountedMsg(event *p2p.PeerEvent) bool
	GetPriceForMsg(event *p2p.PeerEvent) (*big.Int, EntryDirection)
}

//This defines if a price will be debited or credited to an account
type EntryDirection bool

//For some messages the sender pays (e.g. RetrieveRequestMsg),
//for others the receiver (ChunkDeliveryMsg)
const (
	ChargeSender   EntryDirection = true
	ChargeReceiver EntryDirection = false
)

//An accountable message needs some meta information attached to it
//in order to evaluate the correct price
type PriceTag struct {
	Price     *big.Int
	SizeBased bool
	Direction EntryDirection
}

//The default price oracle
type DefaultPriceOracle struct {
	priceMatrix map[string]map[uint64]*PriceTag
}

//Load a default price matrix
func (dpo *DefaultPriceOracle) LoadPriceMatrix() {
	dpo.priceMatrix = dpo.loadDefaultPriceMatrix()
}

//Set up a default price matrix
//This statically builds the price matrix according to previous knowledge
//of which messages need accounting
//The matrix can be queried by
// * the protocol string, returns a sub-map
// * the sub-map by the message code, which returns the PriceTag
func (dpo *DefaultPriceOracle) loadDefaultPriceMatrix() map[string]map[uint64]*PriceTag {
	//setup matrix
	priceMatrix := make(map[string]map[uint64]*PriceTag)

	//define accounted messages from the stream protocol
	streamProtocol := stream.Spec.Name
	priceMatrix[streamProtocol] = make(map[uint64]*PriceTag)

	//get indexes of accounted messages in order to populate the map
	retrieveRequestMsgIndex, ok := stream.Spec.GetCode(stream.RetrieveRequestMsg{})
	if !ok {
		log.Crit("setting up price matrix failed; expected message code not found in Spec!")
		return nil
	}
	deliveryMsgIndex, ok := stream.Spec.GetCode(stream.RetrieveRequestMsg{})
	if !ok {
		log.Crit("setting up price matrix failed; expected message code not found in Spec!")
		return nil
	}

	//assign the price tag to the message
	priceMatrix[streamProtocol][uint64(retrieveRequestMsgIndex)] = &PriceTag{
		Price:     big.NewInt(10),
		SizeBased: false,
		Direction: ChargeSender,
	}

	priceMatrix[streamProtocol][uint64(deliveryMsgIndex)] = &PriceTag{
		Price:     big.NewInt(100),
		SizeBased: true,
		Direction: ChargeReceiver,
	}

	return priceMatrix

}

//Check if a message needs accounting
func (dpo *DefaultPriceOracle) IsAccountedMsg(event *p2p.PeerEvent) bool {
	var protoPriceMap map[uint64]*PriceTag
	var ok bool
	if protoPriceMap, ok = dpo.priceMatrix[event.Protocol]; !ok {
		return false
	}
	if _, ok = protoPriceMap[*event.MsgCode]; !ok {
		return false
	}
	return true
}

//Get the actual price of a message
//If the message is size-based, it returns the calculated price
func (dpo *DefaultPriceOracle) GetPriceForMsg(event *p2p.PeerEvent) (*big.Int, EntryDirection) {
	if dpo.IsAccountedMsg(event) {
		priceTag := dpo.priceMatrix[event.Protocol][*event.MsgCode]
		price := &big.Int{}
		price.Set(priceTag.Price)
		if priceTag.SizeBased {
			price.Mul(priceTag.Price, big.NewInt(int64(*event.MsgSize)))
		}
		return price, priceTag.Direction
	}
	return nil, false
}

//This swap implementation works by listening to message events on the p2p server.
//It then handles the event received, filtering for messages and evaluating
//if it needs accounting
func (s *Swap) registerForEvents(srv *p2p.Server) {
	go func() {
		events := make(chan *p2p.PeerEvent)
		sub := srv.SubscribeEvents(events)
		defer sub.Unsubscribe()

		for {
			select {
			case event := <-events:
				go s.handleMsgEvent(event)
			case err := <-sub.Err():
				log.Error(err.Error())
				return
			}
		}
	}()
}

//Handle the message.
//Determine if it needs accounting, and if yes, account for it
func (s *Swap) handleMsgEvent(event *p2p.PeerEvent) {
	if !s.priceOracle.IsAccountedMsg(event) {
		return
	}
	s.AccountMsgForPeer(event)
}

//Do the accounting
//Depending on the charging type of the message (set in the PriceTag),
//it will charge the sender or receiver
func (s *Swap) AccountMsgForPeer(event *p2p.PeerEvent) {
	price, direction := s.priceOracle.GetPriceForMsg(event)
	if price == nil {
		//TODO what to do in this case? Should not happen
		log.Crit("Price is nil; this should have been accounted for but somehow it failed")
	}

	if direction == ChargeSender {
		//we are sending a ChargeSender message, thus debit us and credit remote
		if event.Type == p2p.PeerEventTypeMsgSend {
			s.chargeLocal(event, price)
			//we are receiving a ChargeSender message, so credit us and debit remote
		} else if event.Type == p2p.PeerEventTypeMsgRecv {
			s.chargeRemote(event, price)
		}
	} else {
		//we are receiving a ChargeReceiver message, thus debit us and credit remote
		if event.Type == p2p.PeerEventTypeMsgRecv {
			s.chargeLocal(event, price)
			//we are sending a ChargeReceiver message, thus credit us and debit remote
		} else if event.Type == p2p.PeerEventTypeMsgSend {
			s.chargeRemote(event, price)
		}
	}
}

//Debit us and credit remote
func (s *Swap) chargeLocal(event *p2p.PeerEvent, amount *big.Int) {
	s.lock.Lock()
	defer s.lock.Unlock()
	//local node is being credited (in its favor), so its balance increases
	if s.peers[event.Peer] == nil {
		s.peers[event.Peer] = &big.Int{}
		s.stateStore.Get(event.Peer.String(), s.peers[event.Peer])
	}
	peerBalance := s.peers[event.Peer]
	peerBalance.Sub(peerBalance, amount)
	//local node is being debited (in favor of remote peer), so its balance decreases
	//TODO: save to store here? init store?
	s.stateStore.Put(event.Peer.String(), peerBalance)
	//(balance *Int) Cmp(payAt)
	// -1 if balance <  payAt
	//  0 if balance == payAt
	// +1 if balance >  payAt
	if peerBalance.Cmp(payAt) == -1 {
		ctx := context.TODO()
		err := s.issueCheque(ctx, event.Peer)
		if err != nil {
			//TODO: special error handling, as at this point the accounting has been done
			//but the cheque could not be sent?
			log.Warn("Payment threshold exceeded, but error sending cheque!", "err", err)
		}
	}
	if peerBalance.Cmp(dropAt) == -1 {
		peer := s.protocol.getPeer(event.Peer)
		if peer != nil {
			peer.Drop(ErrInsufficientFunds)
			//this little hack allows for tests to verify that this error occurred
			event.Error = ErrInsufficientFunds.Error()
		}
	}
	log.Debug(fmt.Sprintf("balance for peer %s: %s", event.Peer.String(), peerBalance.String()))
}

//Credit us and debit remote
func (s *Swap) chargeRemote(event *p2p.PeerEvent, amount *big.Int) {
	s.lock.Lock()
	defer s.lock.Unlock()
	//local node is being credited (in its favor), so its balance increases
	if s.peers[event.Peer] == nil {
		s.peers[event.Peer] = &big.Int{}
	}
	peerBalance := s.peers[event.Peer]
	peerBalance.Add(peerBalance, amount)
	//local node is being credited(in favor of local peer), so its balance increases
	//TODO: save to store here? init store?
	s.stateStore.Put(event.Peer.String(), peerBalance)
	//(balance *Int) Cmp(payAt)
	// -1 if balance <  payAt
	//  0 if balance == payAt
	// +1 if balance >  payAt
	if peerBalance.Cmp(payAt) == -1 {
		ctx := context.TODO()
		err := s.issueCheque(ctx, event.Peer)
		if err != nil {
			//TODO: special error handling, as at this point the accounting has been done
			//but the cheque could not be sent?
			log.Warn("Payment threshold exceeded, but error sending cheque!", "err", err)
		}
	}
	if peerBalance.Cmp(dropAt) == -1 {
		peer := s.protocol.getPeer(event.Peer)
		if peer != nil {
			//this little hack allows for tests to verify that this error occurred
			peer.Drop(ErrInsufficientFunds)
		}
	}
	log.Debug(fmt.Sprintf("balance for peer %s: %s", event.Peer.String(), peerBalance.String()))
}

//get a peer's balance
func (swap *Swap) GetPeerBalance(peer discover.NodeID) *big.Int {
	swap.lock.RLock()
	defer swap.lock.RUnlock()
	if p, ok := swap.peers[peer]; ok {
		return p
	}
	return nil
}

//Issue a cheque for the remote peer. Happens if we are indebted with the peer
//and crossed the payment threshold
func (s *Swap) issueCheque(ctx context.Context, id discover.NodeID) error {
	amount := &big.Int{}
	cheque := s.chequeManager.CreateCheque(id, amount.Abs(payAt))
	msg := IssueChequeMsg{
		Cheque: cheque,
	}
	//TODO: This should now be via the actual SwapProtocol
	p := s.protocol.getPeer(id)
	if p == nil {
		return fmt.Errorf("wanting to send to non-connected peer!")
	}
	return p.Send(ctx, msg)
}

// New - swap constructor
func New(stateStore state.Store) (swap *Swap) {

	priceOracle := &DefaultPriceOracle{}

	swap = &Swap{
		chequeManager: NewChequeManager(stateStore),
		stateStore:    stateStore,
		peers:         make(map[discover.NodeID]*big.Int),
		priceOracle:   priceOracle,
		protocol:      NewProtocol(),
	}
	priceOracle.LoadPriceMatrix()

	return
}
