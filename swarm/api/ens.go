// Copyright 2016 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/logger"
)

// Implements an RPC service
type Ens struct {
	api *Api
}

func NewEns(api *Api) *Ens {
	return &Ens{api}
}

// Resolves ENS domains to contentHashes.
// Could use swarm Api.Resolve if we wanted bzz.resolve both domains and hashes.
func (self *Ens) Resolve(domain string) (string, error) {
	contentHash, err := self.api.dns.Resolve(domain)
	if err != nil {
		err = ErrResolve(err)
		glog.V(logger.Warn).Infof("DNS error : %v", err)
	}
	glog.V(logger.Detail).Infof("host lookup: %v -> %v", err)
	return contentHash.Hex(), err

}
