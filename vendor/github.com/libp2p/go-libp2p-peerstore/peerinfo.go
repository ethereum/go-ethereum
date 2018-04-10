package peerstore

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

// PeerInfo is a small struct used to pass around a peer with
// a set of addresses (and later, keys?). This is not meant to be
// a complete view of the system, but rather to model updates to
// the peerstore. It is used by things like the routing system.
type PeerInfo struct {
	ID    peer.ID
	Addrs []ma.Multiaddr
}

var ErrInvalidAddr = fmt.Errorf("invalid p2p multiaddr")

func InfoFromP2pAddr(m ma.Multiaddr) (*PeerInfo, error) {
	if m == nil {
		return nil, ErrInvalidAddr
	}

	// make sure it's an IPFS addr
	parts := ma.Split(m)
	if len(parts) < 1 {
		return nil, ErrInvalidAddr
	}

	// TODO(lgierth): we shouldn't assume /ipfs is the last part
	ipfspart := parts[len(parts)-1]
	if ipfspart.Protocols()[0].Code != ma.P_IPFS {
		return nil, ErrInvalidAddr
	}

	// make sure the /ipfs value parses as a peer.ID
	peerIdParts := strings.Split(ipfspart.String(), "/")
	peerIdStr := peerIdParts[len(peerIdParts)-1]
	id, err := peer.IDB58Decode(peerIdStr)
	if err != nil {
		return nil, err
	}

	// we might have received just an /ipfs part, which means there's no addr.
	var addrs []ma.Multiaddr
	if len(parts) > 1 {
		addrs = append(addrs, ma.Join(parts[:len(parts)-1]...))
	}

	return &PeerInfo{
		ID:    id,
		Addrs: addrs,
	}, nil
}

func InfoToP2pAddrs(pi *PeerInfo) ([]ma.Multiaddr, error) {
	addrs := []ma.Multiaddr{}
	tpl := "/" + ma.ProtocolWithCode(ma.P_IPFS).Name + "/"
	for _, addr := range pi.Addrs {
		p2paddr, err := ma.NewMultiaddr(tpl + peer.IDB58Encode(pi.ID))
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr.Encapsulate(p2paddr))
	}
	return addrs, nil
}

func (pi *PeerInfo) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"peerID": pi.ID.Pretty(),
		"addrs":  pi.Addrs,
	}
}

func (pi *PeerInfo) MarshalJSON() ([]byte, error) {
	out := make(map[string]interface{})
	out["ID"] = pi.ID.Pretty()
	var addrs []string
	for _, a := range pi.Addrs {
		addrs = append(addrs, a.String())
	}
	out["Addrs"] = addrs
	return json.Marshal(out)
}

func (pi *PeerInfo) UnmarshalJSON(b []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	pid, err := peer.IDB58Decode(data["ID"].(string))
	if err != nil {
		return err
	}
	pi.ID = pid
	addrs, ok := data["Addrs"].([]interface{})
	if ok {
		for _, a := range addrs {
			pi.Addrs = append(pi.Addrs, ma.StringCast(a.(string)))
		}
	}
	return nil
}
