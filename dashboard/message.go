// Copyright 2017 The go-ethereum Authors
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

package dashboard

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/log"
	"time"
)

type Message struct {
	General *GeneralMessage `json:"general,omitempty"`
	Home    *HomeMessage    `json:"home,omitempty"`
	Chain   *ChainMessage   `json:"chain,omitempty"`
	TxPool  *TxPoolMessage  `json:"txpool,omitempty"`
	Network *NetworkMessage `json:"network,omitempty"`
	System  *SystemMessage  `json:"system,omitempty"`
	Logs    *LogsMessage    `json:"logs,omitempty"`
}

type ChartEntries []*ChartEntry

type ChartEntry struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

type GeneralMessage struct {
	Version string `json:"version,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

type HomeMessage struct {
	/* TODO (kurkomisi) */
}

type ChainMessage struct {
	/* TODO (kurkomisi) */
}

type TxPoolMessage struct {
	/* TODO (kurkomisi) */
}

// NetworkMessage contains information about the peers organized based on the IP address.
type NetworkMessage struct {
	Peers *PeersMessage `json:"peers,omitempty"`
}

type PeersMessage struct {
	Bundles          map[string]*PeerBundle `json:"bundles,omitempty"`
	RemovedKnownIP   []string               `json:"removedKnownIP,omitempty"`
	RemovedKnownID   []string               `json:"removedKnownID,omitempty"`
	RemovedUnknownIP []string               `json:"removedUnknownIP,omitempty"`
}

func NewPeersMessage() *PeersMessage {
	return &PeersMessage{
		Bundles: make(map[string]*PeerBundle),
	}
}

func (m *PeersMessage) hasBundle(ip string) bool {
	_, ok := m.Bundles[ip]
	return ok
}

func (m *PeersMessage) hasKnownPeer(ip, id string) bool {
	if m.hasBundle(ip) {
		return m.Bundles[ip].has(id)
	}
	return false
}

func (m *PeersMessage) initBundle(ip string) bool {
	if !m.hasBundle(ip) {
		m.Bundles[ip] = &PeerBundle{
			KnownPeers: make(map[string]*KnownPeer),
		}
		return true
	}
	return false
}

func (m *PeersMessage) initKnownPeer(ip, id string) (bundle, peer bool) {
	return m.initBundle(ip), m.Bundles[ip].initKnownPeer(id)
}

func (m *PeersMessage) getOrInitBundle(ip string) *PeerBundle {
	m.initBundle(ip)
	return m.Bundles[ip]
}

func (m *PeersMessage) getOrInitKnownPeer(ip, id string) *KnownPeer {
	return m.getOrInitBundle(ip).getOrInitKnownPeer(id)
}

func (m *PeersMessage) removeKnownPeer(ip, id string) {
	if b, ok := m.Bundles[ip]; ok {
		b.removeKnownPeer(id)
		if len(b.KnownPeers) < 1 && len(b.UnknownPeers) < 1 {
			delete(m.Bundles, ip)
		}
	}
}

func (m *PeersMessage) removeUnknownPeer(ip string) {
	if b, ok := m.Bundles[ip]; ok {
		if len(b.UnknownPeers) > 0 {
			b.UnknownPeers = b.UnknownPeers[1:]
		}
		if len(b.KnownPeers) < 1 && len(b.UnknownPeers) < 1 {
			delete(m.Bundles, ip)
		}
	}
}

func (m *PeersMessage) clear() {
	for _, bundle := range m.Bundles {
		bundle.Location = nil
		for _, peer := range bundle.KnownPeers {
			peer.clear()
		}
		bundle.UnknownPeers = bundle.UnknownPeers[:0]
	}
	m.RemovedKnownIP = m.RemovedKnownIP[:0]
	m.RemovedKnownID = m.RemovedKnownID[:0]
	m.RemovedUnknownIP = m.RemovedUnknownIP[:0]
}

type PeerBundle struct {
	Location     *GeoLocation          `json:"location,omitempty"` // Geographical location based on IP
	KnownPeers   map[string]*KnownPeer `json:"knownPeers,omitempty"`
	UnknownPeers []*UnknownPeer        `json:"unknownPeers,omitempty"`
}

func (b *PeerBundle) has(id string) bool {
	_, ok := b.KnownPeers[id]
	return ok
}

func (b *PeerBundle) initKnownPeer(id string) bool {
	if !b.has(id) {
		b.KnownPeers[id] = new(KnownPeer)
		return true
	}
	return false
}

func (b *PeerBundle) getOrInitKnownPeer(id string) *KnownPeer {
	b.initKnownPeer(id)
	return b.KnownPeers[id]
}

func (b *PeerBundle) removeKnownPeer(id string) bool {
	if b.has(id) {
		b.KnownPeers[id].clear()
		delete(b.KnownPeers, id)
		return true
	}
	return false
}

type KnownPeer struct {
	Active      bool           `json:"active"`
	Sessions    []*PeerSession `json:"sessions,omitempty"`
	sampleCount int
}

func (peer *KnownPeer) append(session *PeerSession) {
	if session == nil {
		return
	}
	ingress, egress := session.Ingress, session.Egress
	// Truncate the traffic arrays if they have more samples than the limit.
	if first := len(ingress) - sampleLimit; first > 0 {
		ingress = ingress[first:]
	}
	// If the length of the ingress and the egress arrays are different,
	// cut the first part of the longer one. i.e. make sure they have the
	// same length.
	if first := len(ingress) - len(egress); first > 0 {
		ingress = ingress[first:]
	} else if first < 0 {
		egress = egress[-first:]
	}
	if len(peer.Sessions) < 1 {
		// If this is the first session.
		peer.Sessions = append(peer.Sessions, session)
		peer.sampleCount = len(ingress)
		return
	}
	// Cut the old samples from the beginning if the
	// count with the new samples exceeds the limit.
	for l := sampleLimit + len(ingress) - peer.sampleCount; l > 0; l-- {
		for len(peer.Sessions) > 0 && len(peer.Sessions[0].Ingress) < 1 {
			peer.Sessions = peer.Sessions[1:]
		}
		if len(peer.Sessions) < 1 {
			// This can only happen, when the sample count is greater than the
			// sample limit. Theoretically impossible.
			log.Warn("Empty session array with sample count greater than 0")
			return
		}
		first := peer.Sessions[0]
		first.Ingress = first.Ingress[1:]
		first.Egress = first.Egress[1:]
		peer.sampleCount--
	}
	peer.sampleCount += len(ingress)
	if session.Connected != nil {
		peer.Sessions = append(peer.Sessions, session)
		return
	}
	last := peer.Sessions[len(peer.Sessions)-1]
	last.Disconnected = session.Disconnected
	last.Ingress = append(last.Ingress, ingress...)
	last.Egress = append(last.Egress, egress...)
}

func (peer *KnownPeer) upgrade(p *KnownPeer) {
	peer.Active = p.Active
	for _, session := range p.Sessions {
		peer.append(session)
	}
}

func (peer *KnownPeer) clear() {
	for _, s := range peer.Sessions {
		s.Connected = nil
		s.Disconnected = nil
		s.Ingress = nil
		s.Egress = nil
	}
	peer.Sessions = peer.Sessions[:0]
	peer.sampleCount = 0
}

type PeerSession struct {
	Connected    *time.Time   `json:"connected,omitempty"`
	Disconnected *time.Time   `json:"disconnected,omitempty"`
	Ingress      ChartEntries `json:"ingress,omitempty"`
	Egress       ChartEntries `json:"egress,omitempty"`
}

type UnknownPeer struct {
	Connected    time.Time `json:"connected"`
	Disconnected time.Time `json:"disconnected"`
}

// SystemMessage contains the metered system data samples.
type SystemMessage struct {
	ActiveMemory   ChartEntries `json:"activeMemory,omitempty"`
	VirtualMemory  ChartEntries `json:"virtualMemory,omitempty"`
	NetworkIngress ChartEntries `json:"networkIngress,omitempty"`
	NetworkEgress  ChartEntries `json:"networkEgress,omitempty"`
	ProcessCPU     ChartEntries `json:"processCPU,omitempty"`
	SystemCPU      ChartEntries `json:"systemCPU,omitempty"`
	DiskRead       ChartEntries `json:"diskRead,omitempty"`
	DiskWrite      ChartEntries `json:"diskWrite,omitempty"`
}

// LogsMessage wraps up a log chunk. If 'Source' isn't present, the chunk is a stream chunk.
type LogsMessage struct {
	Source *LogFile        `json:"source,omitempty"` // Attributes of the log file.
	Chunk  json.RawMessage `json:"chunk"`            // Contains log records.
}

// LogFile contains the attributes of a log file.
type LogFile struct {
	Name string `json:"name"` // The name of the file.
	Last bool   `json:"last"` // Denotes if the actual log file is the last one in the directory.
}

// Request represents the client request.
type Request struct {
	Logs *LogsRequest `json:"logs,omitempty"`
}

// LogsRequest contains the attributes of the log file the client wants to receive.
type LogsRequest struct {
	Name string `json:"name"` // The request handler searches for log file based on this file name.
	Past bool   `json:"past"` // Denotes whether the client wants the previous or the next file.
}
