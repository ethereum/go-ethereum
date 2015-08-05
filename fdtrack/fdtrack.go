// Copyright 2015 The go-ethereum Authors
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

// Package fdtrack logs statistics about open file descriptors.
package fdtrack

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	mutex sync.Mutex
	all   = make(map[string]int)
)

func Open(desc string) {
	mutex.Lock()
	all[desc] += 1
	mutex.Unlock()
}

func Close(desc string) {
	mutex.Lock()
	defer mutex.Unlock()
	if c, ok := all[desc]; ok {
		if c == 1 {
			delete(all, desc)
		} else {
			all[desc]--
		}
	}
}

func WrapListener(desc string, l net.Listener) net.Listener {
	Open(desc)
	return &wrappedListener{l, desc}
}

type wrappedListener struct {
	net.Listener
	desc string
}

func (w *wrappedListener) Accept() (net.Conn, error) {
	c, err := w.Listener.Accept()
	if err == nil {
		c = WrapConn(w.desc, c)
	}
	return c, err
}

func (w *wrappedListener) Close() error {
	err := w.Listener.Close()
	if err == nil {
		Close(w.desc)
	}
	return err
}

func WrapConn(desc string, conn net.Conn) net.Conn {
	Open(desc)
	return &wrappedConn{conn, desc}
}

type wrappedConn struct {
	net.Conn
	desc string
}

func (w *wrappedConn) Close() error {
	err := w.Conn.Close()
	if err == nil {
		Close(w.desc)
	}
	return err
}

func Start() {
	go func() {
		for range time.Tick(15 * time.Second) {
			mutex.Lock()
			var sum, tracked = 0, []string{}
			for what, n := range all {
				sum += n
				tracked = append(tracked, fmt.Sprintf("%s:%d", what, n))
			}
			mutex.Unlock()
			used, _ := fdusage()
			sort.Strings(tracked)
			glog.Infof("fd usage %d/%d, tracked %d %v", used, fdlimit(), sum, tracked)
		}
	}()
}
