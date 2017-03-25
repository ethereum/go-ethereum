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

// +build windows

package api

import (
	"github.com/ethereum/go-ethereum/log"
)

// Dummy struct and functions to satsfy windows build
type MountInfo struct {
}


func (self *SwarmFS) Mount(mhash, mountpoint string) error {
	log.Info("Platform not supported")
	return nil
}

func (self *SwarmFS) Unmount(mountpoint string) error {
	log.Info("Platform not supported")
	return nil
}

func (self *SwarmFS) Listmounts() (string, error) {
	log.Info("Platform not supported")
	return "",nil
}

func (self *SwarmFS) Stop() error {
	log.Info("Platform not supported")
	return nil
}