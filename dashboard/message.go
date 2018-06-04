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

type ChartEntries []*ChartEntry

type ChartEntry struct {
	Time  time.Time `json:"time,omitempty"`
	Value float64   `json:"value,omitempty"`
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

type LogsMessage struct {
	Old    *LogFile        `json:"old,omitempty"` // Attributes of the log file.
	Chunk  json.RawMessage `json:"chunk"`         // Contains log records.
}

type LogFile struct {
	Name string `json:"name"` // The name of the file.
	Past bool   `json:"past"` // Denotes if the file is the previous or the next one.
	Last bool   `json:"last"` // Denotes if there isn't more file.
}

type Request struct {
	Logs *LogsRequest `json:"logs,omitempty"`
}

type LogsRequest struct {
	Name string `json:"name"` // The request handler searches for log file based on this file name.
	Past bool   `json:"past"` // Denotes whether the client wants the previous or the next file.
}
