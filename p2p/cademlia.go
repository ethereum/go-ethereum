package p2p

import (
	"fmt"
	"sync"
	"time"

	ethlogger "github.com/ethereum/go-ethereum/logger"
)

var cadlogger = ethlogger.NewLogger("CAD")

const (
	hashBits  = 160
	rowLength = 10
)

var maxAge = 180 * time.Nanosecond
var purgeInterval = 300 * time.Second

type Cademlia struct {
	Hash      []byte
	HashBits  int
	RowLength int

	MaxProxSize int

	MaxAge        time.Duration
	PurgeInterval time.Duration

	proxLimit int
	proxSize  int

	rows []*row

	lock  sync.RWMutex
	quitC chan bool
}

func NewCademlia(hash []byte) *Cademlia {
	return &Cademlia{
		Hash: hash, // compulsory fields without default
	}
}

func (self *Cademlia) Start() error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.quitC != nil {
		return nil
	}
	// these + self.Hash can and should be checked against the
	// saved file/db
	if self.HashBits == 0 {
		self.HashBits = hashBits
	}
	if self.RowLength == 0 {
		self.RowLength = rowLength
	}
	// runtime parameters
	if self.MaxProxSize == 0 {
		self.MaxProxSize = self.RowLength
	}
	if self.MaxAge == time.Duration(0) {
		self.MaxAge = maxAge
	}
	if self.PurgeInterval == time.Duration(0) {
		self.PurgeInterval = purgeInterval
	}
	self.rows = make([]*row, self.HashBits)
	for i, _ := range self.rows {
		self.rows[i] = &row{} // will initialise row{int(0),[]*entry(nil),sync.Mutex}
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

func (self *Cademlia) AddPeer(peer peerInfo) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	index := self.DistanceTo(peer.Hash())
	row := self.rows[index]
	added := row.insert(&entry{peer: peer})
	if added {
		if index >= self.proxLimit {
			go self.adjustProx(index, 1)
		}
		cadlogger.Infof("accept peer %x...", peer.Hash()[:8])
	} else {
		err = fmt.Errorf("no worse peer found")
		cadlogger.Infof("reject peer %x..: %v", peer.Hash()[:8], err)
	}
	return
}

func (self *Cademlia) adjustProx(r int, add int) {
	if self.proxSize+add > self.MaxProxSize &&
		self.rows[r].len() > 0 {
		self.proxLimit--
	} else {
		self.proxSize += add
	}
}

func (self *Cademlia) updateProx() {
	var sum, proxSize int
	for _, r := range self.rows {
		sum += r.len()
		if sum <= self.MaxProxSize || r.len() == 0 {
			proxSize = sum
		}
	}
	self.lock.Lock()
	self.proxSize = proxSize
	self.lock.Unlock()
}

func (self *Cademlia) GetPeers(target []byte) (peers []peerInfo) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	index := self.DistanceTo(target)
	var entries []*entry
	if index >= self.proxLimit {
		for i := self.proxLimit; i < self.HashBits; i++ {
			entries = append(entries, self.rows[i].row...)
		}
	} else {
		entries = self.rows[index].row
	}
	for _, entry := range entries {
		peers = append(peers, entry.peer)
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

func (self *row) len() int {
	self.lock.RLock()
	defer self.lock.RUnlock()
	return self.length
}

func (self *row) insert(entry *entry) (added bool) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if len(self.row) >= self.length { // >= allows us to add peers beyond the Rowlength limitation
		worst := self.worst()
		self.row[worst].peer.Disconnect(DiscSubprotocolError)
		self.row[worst] = entry
	} else {
		self.row = append(self.row, entry)
		added = true
		self.length++
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
	self.lock.Lock()
	var newRow []*entry
	for _, entry := range self.row {
		if !entry.peer.LastActive().Before(recently) {
			newRow = append(newRow, entry)
		} else {
			entry.peer.Disconnect(DiscSubprotocolError)
		}
	}
	self.row = newRow
	self.length = len(newRow)
	self.lock.Unlock()
}

func Hash(key []byte) []byte {
	return key
}

func (self *Cademlia) purgeLoop() {
	ticker := time.Tick(self.PurgeInterval)
	for {
		select {
		case <-self.quitC:
			return
		case <-ticker:
			self.lock.Lock()
			for _, r := range self.rows {
				r.purge(time.Now().Add(-self.MaxAge))
			}
			self.updateProx()
			self.lock.Unlock()
		}
	}
}

func Xor(one, other []byte) (xor []byte) {
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return
}

func (self *Cademlia) DistanceTo(other []byte) (ret int) {
	return self.Distance(self.Hash, other)
}

func (self *Cademlia) Distance(one, other []byte) (ret int) {
	xor := Xor(one, other)
	for i := 0; i < self.HashBits; i++ {
		for j := 0; j < 8; j++ {
			if (xor[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return self.HashBits*8 - 1
}
