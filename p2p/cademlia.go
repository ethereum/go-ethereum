package p2p

import (
	"sync"
	"time"

	ethlogger "github.com/ethereum/go-ethereum/logger"
)

var cadlogger = ethlogger.NewLogger("CAD")

const (
	hashBits  = 160
	rowLength = 10
	maxAge    = 1
)

type Cademlia struct {
	hash      []byte
	hashBits  int
	rowLength int
	rows      [hashBits]*row

	depth int

	maxAge        time.Duration
	purgeInterval time.Duration

	lock  sync.RWMutex
	quitC chan bool
}

func newCademlia(hash []byte) *Cademlia {
	return &Cademlia{
		hash:      hash,
		hashBits:  hashBits,
		rowLength: rowLength,
		maxAge:    maxAge * time.Second,
		rows:      [hashBits]*row{},
	}
}

func (self *Cademlia) Start() error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.quitC != nil {
		return nil
	}
	self.quitC = make(chan bool)
	go self.purgeLoop()
	return nil
}

func (self *Cademlia) Stop() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.quitC == nil {
		return
	}
	close(self.quitC)
	self.quitC = nil
}

func (self *Cademlia) AddPeer(peer peerInfo) (needed bool) {
	index := self.commonPrefixLength(peer.Hash())
	row := self.rows[index]
	needed = row.insert(&entry{peer: peer})
	if needed {
		if index >= self.depth {
			go self.updateDepth()
		}
		cadlogger.Infof("accept peer %x...", peer.Hash()[:8])
	} else {
		cadlogger.Infof("reject peer %x... no worse peer found", peer.Hash()[:8])
	}
	return
}

func (self *Cademlia) GetPeers(target []byte) (peers []peerInfo) {
	index := self.commonPrefixLength(target)
	var entries []*entry
	if index >= self.depth {
		for i := self.depth; i < self.hashBits; i++ {
			entries = append(entries, self.rows[i].row...)
		}
	} else {
		entries = self.rows[index].row
	}
	return
}

type entry struct {
	peer peerInfo
	// metadata
}

type row struct {
	length int
	row    []*entry
	lock   sync.RWMutex
}

func (self *row) insert(entry *entry) (ok bool) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if len(self.row) >= self.length {
		worst := self.worst()
		// err = diconnectF(self.row[worst])
		self.row[worst] = entry
	} else {
		self.row = append(self.row, entry)
		ok = true
	}
	return
}

func (self *row) worst() (index int) {
	var oldest time.Time
	for i, entry := range self.row {
		if (oldest == time.Time{}) || entry.peer.LastActive().Before(oldest) {
			oldest = entry.peer.LastActive()
			index = i
		}
	}
	return
}

func (self *row) purge(recently time.Time) {
	var newRow []*entry
	for _, entry := range self.row {
		if !entry.peer.LastActive().Before(recently) {
			newRow = append(newRow, entry)
		} else {
			// self.DisconnectF(entry.peer)
		}
	}
}

func Hash(key []byte) []byte {
	return key
}

func (self *Cademlia) updateDepth() {
}

func (self *Cademlia) purgeLoop() {
	ticker := time.Tick(self.purgeInterval)
	for {
		select {
		case <-self.quitC:
			return
		case <-ticker:
			for _, r := range self.rows {
				r.purge(time.Now().Add(-self.maxAge))
			}
		}
	}
}

func Xor(one, other []byte) (xor []byte) {
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return
}

func (self *Cademlia) commonPrefixLength(other []byte) (ret int) {
	xor := Xor(self.hash, other)
	for i := 0; i < self.hashBits; i++ {
		for j := 0; j < 8; j++ {
			if (xor[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return self.hashBits*8 - 1
}
