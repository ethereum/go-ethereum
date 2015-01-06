package p2p

import (
	"encoding/json"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

type peerInfo interface {
	Addr() *peerAddr
	Hash() []byte
	LastActive() time.Time
	Disconnect(DiscReason)
}

type peerSelector interface {
	AddPeer(peer peerInfo) bool
	GetPeers(target ...[]byte) []peerInfo
	Start() error
	Stop() error
}

type BaseSelector struct {
	DirPath  string
	getPeers func() []peerInfo
	peers    []peerInfo
}

func (self *BaseSelector) AddPeer(peer peerInfo) bool {
	return true
}

func (self *BaseSelector) GetPeers(target ...[]byte) []peerInfo {
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
