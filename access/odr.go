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

// Package access provides a layer to handle local blockchain database and
// on-demand network retrieval
package access

import (
	"errors"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/net/context"
)

var (
	errNoOdr = errors.New("ODR request is not possible")
)

type OdrFunction func(req Request) error

// ChainAccess provides access to blockchain and state data through local
// database and optionally also on-demand network retrieval
type ChainAccess struct {
	db    Database
	odrFn OdrFunction
}

// NewDbChainAccess creates a ChainAccess with ODR disabled
func NewDbChainAccess(db Database) *ChainAccess {
	return NewChainAccess(db, nil)
}

// NewChainAccess create a ChainAccess with optional ODR
func NewChainAccess(db Database, odr OdrFunction) *ChainAccess {
	return &ChainAccess{
		db:    db,
		odrFn: odr,
	}
}

// Db returns the local database assigned to the ChainAccess object
func (self *ChainAccess) Db() Database {
	return self.db
}

// OdrEnabled returns true if this ChainAccess is capable of doing ODR requests
func (self *ChainAccess) OdrEnabled() bool {
	return self.odrFn != nil
}

type Request interface {
	Ctx() context.Context
	StoreResult(db Database)
}

// Retrieve tries to fetch an object from the local db, then from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (self *ChainAccess) Retrieve(req Request) (err error) {
	if !self.OdrEnabled() || !IsOdrContext(req.Ctx()) {
		return errNoOdr
	}
	err = self.odrFn(req)
	if err == nil {
		// retrieved from network, store in db
		req.StoreResult(self.Db())
	} else {
		glog.V(logger.Info).Infof("ODR retrieve err = %v", err)
	}
	return
}
