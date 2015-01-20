package p2p

import (
	"fmt"
	"math"
	"sync"
	"time"

	ethlogger "github.com/ethereum/go-ethereum/logger"
)

var cadlogger = ethlogger.NewLogger("CAD")

const (
	hashBytes = 20
	rowLength = 10
	maxProx   = 20
)

var maxAge = 180 * time.Nanosecond
var purgeInterval = 300 * time.Second

type Cademlia struct {
	Hash      []byte
	HashBytes int
	RowLength int

	MaxProx        int
	MaxProxBinSize int

	MaxAge        time.Duration
	PurgeInterval time.Duration

	proxLimit int
	proxSize  int

	rows []*row

	lock  sync.RWMutex
	quitC chan bool
}

// public constructor with compulsory arguments
// hash is a byte slice of length equal to self.HashBytes
func NewCademlia(hash []byte) *Cademlia {
	return &Cademlia{
		Hash: hash, // compulsory fields without default
	}
}

// Start brings up a pool of peers potentially from an offline persisted source
// and sets default values for optional parameters
func (self *Cademlia) Start() error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.quitC != nil {
		return nil
	}
	// these + self.Hash can and should be checked against the
	// saved file/db
	if self.HashBytes == 0 {
		self.HashBytes = hashBytes
	}
	if self.MaxProx == 0 {
		self.MaxProx = maxProx
	}
	if self.RowLength == 0 {
		self.RowLength = rowLength
	}
	// runtime parameters
	if self.MaxProxBinSize == 0 {
		self.MaxProxBinSize = self.RowLength
	}
	if self.MaxAge == time.Duration(0) {
		self.MaxAge = maxAge
	}
	if self.PurgeInterval == time.Duration(0) {
		self.PurgeInterval = purgeInterval
	}
	self.rows = make([]*row, self.MaxProx)
	for i, _ := range self.rows {
		self.rows[i] = &row{} // will initialise row{int(0),[]*entry(nil),sync.Mutex}
	}
	self.quitC = make(chan bool)
	go self.purgeLoop()
	return nil
}

// Stop saves the routing table into a persistant form
func (self *Cademlia) Stop() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.quitC == nil {
		return
	}
	close(self.quitC)
	self.quitC = nil
}

// AddPeer is the entry point where new peers are suggested for addition to the peer pool
// peers conform to the peerrInfo interface
// AddPeer(peer) returns an error if it deems the peer unworthy
func (self *Cademlia) AddPeer(peer peerInfo) (err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	index := self.ProximityBin(peer.Hash())
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

// adjust Prox (proxLimit and proxSize after an insertion of add entries into row r)
func (self *Cademlia) adjustProx(r int, add int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if r >= self.proxLimit &&
		self.proxSize+add > self.MaxProxBinSize &&
		self.rows[r].len() > 0 {
		self.proxLimit++
	} else {
		self.proxSize += add
	}
}

// updates Prox (proxLimit and proxSize after purging entries)
func (self *Cademlia) updateProx() {
	self.lock.Lock()
	defer self.lock.Unlock()
	var sum int
	for i := self.MaxProx - 1; i >= 0; i-- {
		l := self.rows[i].len()
		sum += l
		if sum <= self.MaxProxBinSize || l == 0 {
			self.proxSize = sum
		}
	}
}

// GetPeers(target) returns the list of peers belonging to the same proximity bin as the target. The most proximate bin will be the union of the bins between proxLimit and MaxProx. proxLimit is dynamically adjusted so that 1) there is no empty rows in bin < proxLimit and 2) the sum of all items are the maximum possible but lower than MaxProxBinSize
func (self *Cademlia) GetPeers(target []byte) (peers []peerInfo) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	index := self.ProximityBin(target)
	var entries []*entry
	if index >= self.proxLimit {
		for i := self.proxLimit; i < self.MaxProx; i++ {
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

// entry wrapper type for peer object adding potentially persisted metadata for offline permanent record
type entry struct {
	peer peerInfo
	// metadata
}

// in situ mutable row
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

// insert adds a peer to a row either by appending to existing items if row length does not exceed RowLength, or by replacing the worst entry in the row
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

// worst expunges the single worst entry in a row, where worst entry is with a peer that has not been active the longests
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

// expunges entries from a row that were last active more that MaxAge ago
// calls Disconnect on entry.peer
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

/*
Taking the proximity value relative to a fix point x classifies the points in the space (n byte long byte sequences) into bins the items in which are each at most half as distant from x as items in the previous bin. Given a sample of uniformly distrbuted items (a hash function over arbitrary sequence) the proximity scale maps onto series of subsets with cardinalities on a negative exponential scale.

It also has the property that any two item belonging to the same bin are at most half as distant from each other as they are from x.

If we think of random sample of items in the bins as connections in a network of interconnected nodes than relative proximity can serve as the basis for local decisions for graph traversal where the task is to find a route between two points. Since in every step of forwarding, the finite distance halves, there is a guaranteed constant maximum limit on the number of hops needed to reach one node from the other.
*/

func (self *Cademlia) ProximityBin(other []byte) (ret int) {
	return int(math.Min(float64(self.MaxProx), float64(self.Proximity(self.Hash, other))))
}

/*
The distance metric MSB(x, y) of two equal length bytesequences x an y is the value of the
binary integer cast of the xor-ed bytesequence most significant bit first.
Proximity(x, y) counts the common zeros in the front of this distance measure.
*/

func (self *Cademlia) Proximity(one, other []byte) (ret int) {
	xor := Xor(one, other)
	for i := 0; i < self.HashBytes; i++ {
		for j := 0; j < 8; j++ {
			if (xor[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return self.HashBytes*8 - 1
}

func Xor(one, other []byte) (xor []byte) {
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return
}
