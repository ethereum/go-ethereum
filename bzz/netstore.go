package bzz

/*
DHT implements the chunk store that directly communicates with the bzz protocol on the one hand and the kademlia node table on the other.
It does forwarding for incoming requests and handles expiry/timeout.
*/

type peerPool interface {
	GetPeers(target Key, peers []peer)
}

// it implements the ChunkStore interface
type netStore struct {
	peerPool peerPool
	// cademlia
}

func (self *DPA) Put(chunk *Chunk) {

	return
}

func (self *DPA) Get(key Key) (chunk *Chunk, err error) {
	return
}
