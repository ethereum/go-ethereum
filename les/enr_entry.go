// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"time"

	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

// lesEntry is the "les" ENR entry. This is set for LES servers only.
type lesEntry struct {
	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e lesEntry) ENRKey() string {
	return "les"
}

// setupDiscovery creates the node discovery source for the eth protocol.
func (eth *LightEthereum) setupDiscovery(cfg *p2p.Config) (enode.Iterator, error) {
	it := enode.NewFairMix(0)
	if len(eth.config.EthDiscoveryURLs) != 0 {
		client := dnsdisc.NewClient(dnsdisc.Config{})
		dns, err := client.NewIterator(eth.config.EthDiscoveryURLs...)
		if err != nil {
			return nil, err
		}
		it.AddSource(dns)
	}
	if cfg.DiscoveryV5 && eth.p2pServer.DiscV5 != nil {
		it.AddSource(eth.p2pServer.DiscV5.RandomNodes())
	}
	forkFilter := forkid.NewFilter(eth.blockchain)
	var delay time.Duration
	return enode.Filter(it, func(n *enode.Node) bool {
		var (
			les struct {
				_ []rlp.RawValue `rlp:"tail"`
			}
			eth struct {
				ForkID forkid.ID
				_      []rlp.RawValue `rlp:"tail"`
			}
		)
		if n.Load(enr.WithEntry("les", &les)) == nil && n.Load(enr.WithEntry("eth", &eth)) == nil && forkFilter(eth.ForkID) == nil {
			delay /= 2
			return true
		}
		delay += time.Millisecond
		time.Sleep(delay)
		return false
	}), nil
}
