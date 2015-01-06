package p2p

import (
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

type peerInfo interface {
	Addr() *peerAddr
	Hash() []byte
	// Pubkey() []byte
	LastActive() time.Time
	Disconnect() error
	Connect() error
}

type peerSelector interface {
	AddPeer(peer peerInfo) error
	GetPeers(target ...[]byte) []*peerAddr
	Start() error
	Stop() error
}

type BaseSelector struct {
	DirPath  string
	getPeers func() []*peerAddr
	peers    []peerInfo
}

func (self *BaseSelector) AddPeer(peer peerInfo) error {
	return nil
}

func (self *BaseSelector) GetPeers(target ...[]byte) []*peerAddr {
	return self.getPeers()
}

func (self *BaseSelector) Start() error {
	if len(self.DirPath) > 0 {
		path := path.Join(self.DirPath, "peers.json")
		peers, err := ReadPeers(path)
		if err != nil {
			return err
		}
		self.peers = peers
	}
	return nil
}

func (self *BaseSelector) Stop() error {
	if len(self.DirPath) > 0 {
		path := path.Join(self.DirPath, "peers.json")
		if err := WritePeers(path, self.peers); err != nil {
			return err
		}
	}
	return nil
}

func WritePeers(path string, addresses []peerInfo) error {
	if len(addresses) > 0 {
		data, err := json.MarshalIndent(addresses, "", "    ")
		if err == nil {
			ethutil.WriteFile(path, data)
		}
		return err
	}
	return nil
}

func ReadPeers(path string) (peers []peerInfo, err error) {
	var data string
	data, err = ethutil.ReadAllFile(path)
	if err == nil {
		json.Unmarshal([]byte(data), &peers)
	}
	return
}

const (
	hashBits  = 160
	rowLength = 10
	maxAge    = 1
)

type Cademlia struct {
	hash      []byte
	hashBits  int
	rowLength int
	// index     map[string]peerInfo
	rows [hashBits]*row

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
		// index:     make(map[string]peerInfo),
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

func (self *Cademlia) AddPeer(peer peerInfo) (err error) {
	index := self.commonPrefixLength(peer.Hash())
	row := self.rows[index]
	needed := row.insert(&entry{peer: peer})
	if needed {
		if index >= self.depth {
			self.updateDepth()
		}
	} else {
		err = fmt.Errorf("no worse peer found")
	}
	return
}

func (self *Cademlia) GetPeers(target []byte) (peers []*peerAddr) {
	index := self.commonPrefixLength(target)
	var entries []*entry
	if index >= self.depth {
		for i := self.depth; i < self.hashBits; i++ {
			entries = append(entries, self.rows[i].row...)
		}
	} else {
		entries = self.rows[index].row
	}

	for _, entry := range entries {
		peers = append(peers, entry.peer.Addr())
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
		self.row[self.worst()] = entry
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
			entry.peer.Disconnect()
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
