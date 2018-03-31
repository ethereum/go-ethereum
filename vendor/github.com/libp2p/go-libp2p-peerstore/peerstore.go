package peerstore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ic "github.com/libp2p/go-libp2p-crypto"

	//ds "github.com/jbenet/go-datastore"
	//dssync "github.com/jbenet/go-datastore/sync"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("peerstore")

const (
	// AddressTTL is the expiration time of addresses.
	AddressTTL = time.Hour
)

// Peerstore provides a threadsafe store of Peer related
// information.
type Peerstore interface {
	AddrBook
	KeyBook
	Metrics

	// Peers returns a list of all peer.IDs in this Peerstore
	Peers() []peer.ID

	// PeerInfo returns a peer.PeerInfo struct for given peer.ID.
	// This is a small slice of the information Peerstore has on
	// that peer, useful to other services.
	PeerInfo(peer.ID) PeerInfo

	// Get/Put is a simple registry for other peer-related key/value pairs.
	// if we find something we use often, it should become its own set of
	// methods. this is a last resort.
	Get(id peer.ID, key string) (interface{}, error)
	Put(id peer.ID, key string, val interface{}) error

	GetProtocols(peer.ID) ([]string, error)
	AddProtocols(peer.ID, ...string) error
	SetProtocols(peer.ID, ...string) error
	SupportsProtocols(peer.ID, ...string) ([]string, error)
}

// AddrBook is an interface that fits the new AddrManager. I'm patching
// it up in here to avoid changing a ton of the codebase.
type AddrBook interface {

	// AddAddr calls AddAddrs(p, []ma.Multiaddr{addr}, ttl)
	AddAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration)

	// AddAddrs gives AddrManager addresses to use, with a given ttl
	// (time-to-live), after which the address is no longer valid.
	// If the manager has a longer TTL, the operation is a no-op for that address
	AddAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration)

	// SetAddr calls mgr.SetAddrs(p, addr, ttl)
	SetAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration)

	// SetAddrs sets the ttl on addresses. This clears any TTL there previously.
	// This is used when we receive the best estimate of the validity of an address.
	SetAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration)

	// UpdateAddrs updates the addresses associated with the given peer that have
	// the given oldTTL to have the given newTTL.
	UpdateAddrs(p peer.ID, oldTTL time.Duration, newTTL time.Duration)

	// Addresses returns all known (and valid) addresses for a given peer
	Addrs(p peer.ID) []ma.Multiaddr

	// AddrStream returns a channel that gets all addresses for a given
	// peer sent on it. If new addresses are added after the call is made
	// they will be sent along through the channel as well.
	AddrStream(context.Context, peer.ID) <-chan ma.Multiaddr

	// ClearAddresses removes all previously stored addresses
	ClearAddrs(p peer.ID)
}

// KeyBook tracks the Public keys of Peers.
type KeyBook interface {
	PubKey(peer.ID) ic.PubKey
	AddPubKey(peer.ID, ic.PubKey) error

	PrivKey(peer.ID) ic.PrivKey
	AddPrivKey(peer.ID, ic.PrivKey) error
}

type keybook struct {
	pks map[peer.ID]ic.PubKey
	sks map[peer.ID]ic.PrivKey

	sync.RWMutex // same lock. wont happen a ton.
}

func newKeybook() *keybook {
	return &keybook{
		pks: map[peer.ID]ic.PubKey{},
		sks: map[peer.ID]ic.PrivKey{},
	}
}

func (kb *keybook) Peers() []peer.ID {
	kb.RLock()
	ps := make([]peer.ID, 0, len(kb.pks)+len(kb.sks))
	for p := range kb.pks {
		ps = append(ps, p)
	}
	for p := range kb.sks {
		if _, found := kb.pks[p]; !found {
			ps = append(ps, p)
		}
	}
	kb.RUnlock()
	return ps
}

func (kb *keybook) PubKey(p peer.ID) ic.PubKey {
	kb.RLock()
	pk := kb.pks[p]
	kb.RUnlock()
	return pk
}

func (kb *keybook) AddPubKey(p peer.ID, pk ic.PubKey) error {

	// check it's correct first
	if !p.MatchesPublicKey(pk) {
		return errors.New("ID does not match PublicKey")
	}

	kb.Lock()
	kb.pks[p] = pk
	kb.Unlock()
	return nil
}

func (kb *keybook) PrivKey(p peer.ID) ic.PrivKey {
	kb.RLock()
	sk := kb.sks[p]
	kb.RUnlock()
	return sk
}

func (kb *keybook) AddPrivKey(p peer.ID, sk ic.PrivKey) error {

	if sk == nil {
		return errors.New("sk is nil (PrivKey)")
	}

	// check it's correct first
	if !p.MatchesPrivateKey(sk) {
		return errors.New("ID does not match PrivateKey")
	}

	kb.Lock()
	kb.sks[p] = sk
	kb.Unlock()
	return nil
}

type peerstore struct {
	*keybook
	*metrics
	AddrManager

	// store other data, like versions
	//ds ds.ThreadSafeDatastore
	// TODO: use a datastore for this
	ds     map[string]interface{}
	dslock sync.Mutex

	// lock for protocol information, separate from datastore lock
	protolock sync.Mutex
}

// NewPeerstore creates a threadsafe collection of peers.
func NewPeerstore() Peerstore {
	return &peerstore{
		keybook:     newKeybook(),
		metrics:     NewMetrics(),
		AddrManager: AddrManager{},
		//ds:          dssync.MutexWrap(ds.NewMapDatastore()),
		ds: make(map[string]interface{}),
	}
}

func (ps *peerstore) Put(p peer.ID, key string, val interface{}) error {
	//dsk := ds.NewKey(string(p) + "/" + key)
	//return ps.ds.Put(dsk, val)
	ps.dslock.Lock()
	defer ps.dslock.Unlock()
	ps.ds[string(p)+"/"+key] = val
	return nil
}

var ErrNotFound = errors.New("item not found")

func (ps *peerstore) Get(p peer.ID, key string) (interface{}, error) {
	//dsk := ds.NewKey(string(p) + "/" + key)
	//return ps.ds.Get(dsk)

	ps.dslock.Lock()
	defer ps.dslock.Unlock()
	i, ok := ps.ds[string(p)+"/"+key]
	if !ok {
		return nil, ErrNotFound
	}
	return i, nil
}

func (ps *peerstore) Peers() []peer.ID {
	set := map[peer.ID]struct{}{}
	for _, p := range ps.keybook.Peers() {
		set[p] = struct{}{}
	}
	for _, p := range ps.AddrManager.Peers() {
		set[p] = struct{}{}
	}

	pps := make([]peer.ID, 0, len(set))
	for p := range set {
		pps = append(pps, p)
	}
	return pps
}

func (ps *peerstore) PeerInfo(p peer.ID) PeerInfo {
	return PeerInfo{
		ID:    p,
		Addrs: ps.AddrManager.Addrs(p),
	}
}

func (ps *peerstore) SetProtocols(p peer.ID, protos ...string) error {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()

	protomap := make(map[string]struct{})
	for _, proto := range protos {
		protomap[proto] = struct{}{}
	}

	return ps.Put(p, "protocols", protomap)
}

func (ps *peerstore) AddProtocols(p peer.ID, protos ...string) error {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()
	protomap, err := ps.getProtocolMap(p)
	if err != nil {
		return err
	}

	for _, proto := range protos {
		protomap[proto] = struct{}{}
	}

	return ps.Put(p, "protocols", protomap)
}

func (ps *peerstore) getProtocolMap(p peer.ID) (map[string]struct{}, error) {
	iprotomap, err := ps.Get(p, "protocols")
	switch err {
	default:
		return nil, err
	case ErrNotFound:
		return make(map[string]struct{}), nil
	case nil:
		cast, ok := iprotomap.(map[string]struct{})
		if !ok {
			return nil, fmt.Errorf("stored protocol set was not a map")
		}

		return cast, nil
	}
}

func (ps *peerstore) GetProtocols(p peer.ID) ([]string, error) {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()
	pmap, err := ps.getProtocolMap(p)
	if err != nil {
		return nil, err
	}

	var out []string
	for k, _ := range pmap {
		out = append(out, k)
	}

	return out, nil
}

func (ps *peerstore) SupportsProtocols(p peer.ID, protos ...string) ([]string, error) {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()
	pmap, err := ps.getProtocolMap(p)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, proto := range protos {
		if _, ok := pmap[proto]; ok {
			out = append(out, proto)
		}
	}

	return out, nil
}

func PeerInfos(ps Peerstore, peers []peer.ID) []PeerInfo {
	pi := make([]PeerInfo, len(peers))
	for i, p := range peers {
		pi[i] = ps.PeerInfo(p)
	}
	return pi
}

func PeerInfoIDs(pis []PeerInfo) []peer.ID {
	ps := make([]peer.ID, len(pis))
	for i, pi := range pis {
		ps[i] = pi.ID
	}
	return ps
}
