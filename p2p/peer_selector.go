package p2p

import (
	"time"
)

type PeerSelector interface {
	SuggestPeer(addr *peerAddr) (ok bool)
	// AddPeer(addr *peerAddr) (ok bool)
	GetPeers(target []byte) []*peerAddr
	Start()
	Stop()
}

type BaseSelector struct {
	DirPath string
}

func (self *BaseSelector) SuggestPeer(addr *peerAddr) bool {
	return true
}

const (
	hashBits  = 160
	rowLength = 10
	maxAge    = 1
)

type Cademlia struct {
	rows      [hashBits]*row
	hashBits  int
	rowLength int
	maxAge    time.Duration
	index     map[string]*peerData
}

type row struct {
	length int
	row    []*peerData
	lock   sync.RWMutex
}

func (self *row) addresses() (addrs []*peerAddr) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	for _, p := range self.row {
		addrs = append(addrs, p.addr)
	}
	return
}

func (self *row) insert(addr *peerAddr) (ok bool) {
	self.lock.Lock()
	defer self.lock.Unlock()
	peerData := &peerData{addr: addr}
	if len(self.row) >= self.length {
		self.row[self.worst()] = peerData
	} else {
		self.row = append(self.row, peerData)
		ok = true
	}
	return
}

func (self *row) worst() (index int) {
	var oldest time.Time
	for i, p := range self.row {
		if oldest == nil || p.addr.LastSeen().Before(oldest) {
			oldest = p.addr.LastSeen
			index = i
		}
	}
	return
}

func (self *row) purge(maxAge time.Time) {
	var newRow []*peerData
	for _, p := range self.row {
		if !p.addr.LastSeen().Before(maxAge) {
			newRow = append(newRow, p)
		}
	}
}

type peerData struct {
	addr *peerAddr
	hash []byte
}

func Hash([]byte) []byte {

}

func (self *Cademlia) prefixLength(other []byte) {

}

func newCademlia() *Cademlia {
	return &Cademlia{
		hashBits:  hashBits,
		rowLength: rowLength,
		maxAge:    maxAge * time.Second,
		rows:      make([hashBits]*row),
		index:     make(map[string]*peerData),
	}
}

func (self *Cademlia) Start() {
	go self.purgeLoop()
}

func (self *Cademlia) purgeLoop() {
	ticker := time.Tick(self.purgeInterval)
	for {
		select {
		case <-ticker:
			for _, r := range self.rows {
				r.purge(time.Since(self.maxAge))
			}
		}
	}
}

func (self *Cademlia) SuggestPeer(addr *peerAddr) bool {
	index := self.commonPrefixLength(Hash(addr.Pubkey))
	row := self.rows[index]
	longer := row.insert(addr)
	if index >= self.depth && longer {
		self.updateDepth()
	}
	return longer
}

func (self *Cademlia) GetPeers(target []byte) (peers []*peerAddr) {
	index := self.prefixLength(target)
	if index >= self.depth {
		for i := self.depth; i < hashBits; i++ {
			peers = append(peers, self.rows[i].addresses())
		}
	} else {
		peers = self.rows[index].addresses()
	}
	return
}

func Xor(one, other []byte) (xor []byte) {
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return
}

func (self *Cademlia) commonPrefixLength(other []byte) (ret int) {
	xor := Xor(self.hash, other)
	for i := 0; i < len(self.hash); i++ {
		for j := 0; j < 8; j++ {
			if (xor[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return len(self.hash)*8 - 1
}
