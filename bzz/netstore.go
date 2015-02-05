package bzz

/*
DHT implements the chunk store that directly communicates with the bzz protocol on the one hand and the kademlia node table on the other.
It does forwarding for incoming requests and handles expiry/timeout.
*/

import (
	"math/rand"
	"time"
)

// This is a mock implementation with a fixed peer pool with no distinction between peers
type peerPool struct {
	pool map[string]peer
}

func (self *peerPool) addPeer(p peer) {
	self.pool[p.peer.identity.Pubkey()] = p
}

func (self *peerPool) removePeer(p peer) {
	delete(self.pool, p.peer.identity.Pubkey)
}

func (self *peerPool) GetPeers(target Key) (peers []peer) {
	for key, value := range self.pool {
		peers = append(peers, value)
	}
	return
}

// it implements the ChunkStore interface
type netStore struct {
	peerPool peerPool
	// cademlia
}

func (self *netStore) Put(chunk *Chunk) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	req := storeRequestMsgData{
		Key:  chunk.Key,
		Data: chunk.Data,
		Id:   r.Int63(),
		Size: chunk.Size,
	}
	for _, peer := range self.peerPool.GetPeers(chunk.Key) {
		go peer.store(req)
	}
	return
}

func (self *DPA) Get(key Key) (chunk *Chunk, err error) {
	return
}
