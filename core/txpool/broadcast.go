package txpool

import (
	"bytes"
	"sync"
)

type Broadcast struct {
	allClient        map[*Client]bool
	broadcastMessage chan []byte
	registerClient   chan *Client
	unregisterClient chan *Client
	last5Msg         [][]byte
	mux              sync.RWMutex
}

func NewBroadcast() *Broadcast {
	return &Broadcast{
		allClient:        make(map[*Client]bool),
		broadcastMessage: make(chan []byte),
		registerClient:   make(chan *Client),
		unregisterClient: make(chan *Client),
		last5Msg:         [][]byte{nil, nil, nil, nil, nil},
	}
}
func (b *Broadcast) Run() {
	for {
		select {
		case newClient := <-b.registerClient:
			b.allClient[newClient] = true
		case clientData := <-b.unregisterClient:
			_, stateClient := b.allClient[clientData]
			if stateClient {
				delete(b.allClient, clientData)
				close(clientData.sendMessage)
			}
		case messageData := <-b.broadcastMessage:
			if b.check(messageData) {
				for clientData := range b.allClient {
					clientData.sendMessage <- messageData
				}
			}
		}
	}
}

func (b *Broadcast) check(msg []byte) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	for _, m := range b.last5Msg {
		if bytes.Equal(m, msg) {
			return false
		}
	}
	b.last5Msg = append(b.last5Msg[1:], msg)
	return true
}
