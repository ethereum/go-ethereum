// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	VersionMajor = 1        // Major version component of the current release
	VersionMinor = 5        // Minor version component of the current release
	VersionPatch = 3        // Patch version component of the current release
	VersionMeta  = "stable" // Version metadata to append to the version string
)

// Version holds the textual version string.
var Version = func() string {
	v := fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

// MakeDefaultExtraData returns the default Ethereum block extra data blob.
func MakeDefaultExtraData(clientIdentifier string) []byte {
	var clientInfo = struct {
		Version   uint
		Name      string
		GoVersion string
		Os        string
	}{uint(VersionMajor<<16 | VersionMinor<<8 | VersionPatch), clientIdentifier, runtime.Version(), runtime.GOOS}
	extra, err := rlp.EncodeToBytes(clientInfo)
	if err != nil {
		glog.V(logger.Warn).Infoln("error setting canonical miner information:", err)
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize.Uint64() {
		glog.V(logger.Warn).Infoln("error setting canonical miner information: extra exceeds", params.MaximumExtraDataSize)
		glog.V(logger.Debug).Infof("extra: %x\n", extra)
		return nil
	}
	return extra
}
