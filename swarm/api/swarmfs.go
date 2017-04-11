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

package api

import (
	"sync"
	"time"
)

const (
	Swarmfs_Version = "0.1"
	mountTimeout    = time.Second * 5
	maxFuseMounts   = 5
)

type SwarmFS struct {
	swarmApi     *Api
	activeMounts map[string]*MountInfo
	activeLock   *sync.RWMutex
}

func NewSwarmFS(api *Api) *SwarmFS {
	swarmfs := &SwarmFS{
		swarmApi:     api,
		activeLock:   &sync.RWMutex{},
		activeMounts: map[string]*MountInfo{},
	}
	return swarmfs
}
