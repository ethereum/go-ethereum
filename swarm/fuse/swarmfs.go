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

package fuse

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/swarm/api"
)

const (
	Swarmfs_Version = "0.1"
	mountTimeout    = time.Second * 5
	unmountTimeout  = time.Second * 10
	maxFuseMounts   = 5
)

var (
	swarmfs     *SwarmFS // Swarm file system singleton
	swarmfsLock sync.Once

	inode     uint64 = 1 // global inode
	inodeLock sync.RWMutex
)

type SwarmFS struct {
	swarmApi     *api.Api
	activeMounts map[string]*MountInfo
	swarmFsLock  *sync.RWMutex
}

func NewSwarmFS(api *api.Api) *SwarmFS {
	swarmfsLock.Do(func() {
		swarmfs = &SwarmFS{
			swarmApi:     api,
			swarmFsLock:  &sync.RWMutex{},
			activeMounts: map[string]*MountInfo{},
		}
	})
	return swarmfs

}

// Inode numbers need to be unique, they are used for caching inside fuse
func NewInode() uint64 {
	inodeLock.Lock()
	defer inodeLock.Unlock()
	inode += 1
	return inode
}
