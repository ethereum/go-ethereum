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

func (m *Message) DeepCopy() *Message {
	return &Message{
		m.General.DeepCopy(),
		m.Home,
		m.Chain,
		m.TxPool,
		m.Network,
		m.System.DeepCopy(),
		m.Logs,
	}
}

type ChartEntries []*ChartEntry

func (ce ChartEntries) DeepCopy() ChartEntries {
	nce := make(ChartEntries, len(ce))
	for i, v := range ce {
		nce[i] = v.DeepCopy()
	}
	return nce
}

type ChartEntry struct {
	Time  time.Time `json:"time,omitempty"`
	Value float64   `json:"value,omitempty"`
}

func (ce *ChartEntry) DeepCopy() *ChartEntry {
	return &ChartEntry{ce.Time, ce.Value}
}

type GeneralMessage struct {
	Version string `json:"version,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

func (m *GeneralMessage) DeepCopy() *GeneralMessage {
	return &GeneralMessage{m.Version, m.Commit}
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

type NetworkMessage struct {
	/* TODO (kurkomisi) */
}

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

func (m *SystemMessage) DeepCopy() *SystemMessage {
	return &SystemMessage{
		m.ActiveMemory.DeepCopy(),
		m.VirtualMemory.DeepCopy(),
		m.NetworkIngress.DeepCopy(),
		m.NetworkEgress.DeepCopy(),
		m.ProcessCPU.DeepCopy(),
		m.SystemCPU.DeepCopy(),
		m.DiskRead.DeepCopy(),
		m.DiskWrite.DeepCopy(),
	}
}

type LogsMessage struct {
	Chunk json.RawMessage `json:"chunk,omitempty"`
}

type Request struct {
	Logs *LogsRequest `json:"logs,omitempty"`
}

type LogsRequest struct {
	Time time.Time `json:"time,omitempty"`
}
