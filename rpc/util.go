/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
package rpc

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
)

var rpclogger = logger.NewLogger("RPC")

type Log struct {
	Address string   `json:"address"`
	Topic   []string `json:"topic"`
	Data    string   `json:"data"`
	Number  uint64   `json:"number"`
}

func toLogs(logs state.Logs) (ls []Log) {
	ls = make([]Log, len(logs))

	for i, log := range logs {
		var l Log
		l.Topic = make([]string, len(log.Topics()))
		l.Address = log.Address().Hex()
		l.Data = common.ToHex(log.Data())
		l.Number = log.Number()
		for j, topic := range log.Topics() {
			l.Topic[j] = topic.Hex()
		}
		ls[i] = l
	}

	return
}

type whisperFilter struct {
	messages []xeth.WhisperMessage
	timeout  time.Time
	id       int
}

func (w *whisperFilter) add(msgs ...xeth.WhisperMessage) {
	w.messages = append(w.messages, msgs...)
}
func (w *whisperFilter) get() []xeth.WhisperMessage {
	w.timeout = time.Now()
	tmp := w.messages
	w.messages = nil
	return tmp
}

type logFilter struct {
	logs    state.Logs
	timeout time.Time
	id      int
}

func (l *logFilter) add(logs ...state.Log) {
	l.logs = append(l.logs, logs...)
}

func (l *logFilter) get() state.Logs {
	l.timeout = time.Now()
	tmp := l.logs
	l.logs = nil
	return tmp
}
