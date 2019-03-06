package les

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type ulc struct {
	trustedKeys        map[string]struct{}
	minTrustedFraction int
}

func newULC(ulcConfig *eth.ULCConfig) *ulc {
	if ulcConfig == nil {
		return nil
	}

	m := make(map[string]struct{}, len(ulcConfig.TrustedServers))
	for _, id := range ulcConfig.TrustedServers {
		node, err := enode.ParseV4(id)
		if err != nil {
			fmt.Println("node:", id, " err:", err)
			continue
		}
		m[node.ID().String()] = struct{}{}
	}

	return &ulc{m, ulcConfig.MinTrustedFraction}
}

func (u *ulc) isTrusted(p enode.ID) bool {
	if u.trustedKeys == nil {
		return false
	}
	_, ok := u.trustedKeys[p.String()]
	return ok
}
