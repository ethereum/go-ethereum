// Copyright 2018 The go-ethereum Authors
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

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"time"
)

// The TimedExternalAPI implements ExternalAPI, but can be configured to time out after a specified interval.
// This can be used to ensure that callers get a response within reasonable time,
// even if the user is unresponsive.
type TimedExternalAPI struct {
	timeout time.Duration
	next    ExternalAPI
}

func NewTimedExternalAPI(next ExternalAPI, timeout time.Duration) ExternalAPI {
	return &TimedExternalAPI{timeout, next}
}

var (
	ErrTimeout = fmt.Errorf("timeout occurred")
)

func (t *TimedExternalAPI) List(ctx context.Context) ([]common.Address, error) {
	type response struct {
		addr []common.Address
		err  error
	}
	ch := make(chan response)
	go func() {
		addr, err := t.next.List(ctx)
		ch <- response{addr, err}
	}()

	select {
	case r := <-ch:
		return r.addr, r.err
	case <-time.After(t.timeout):
		log.Info("timeout", "op", "list")
		go func() { <-ch }()
		return []common.Address{}, ErrTimeout
	}
}

func (t *TimedExternalAPI) New(ctx context.Context) (accounts.Account, error) {
	type response struct {
		acc accounts.Account
		err error
	}
	ch := make(chan response)
	go func() {
		acc, err := t.next.New(ctx)
		ch <- response{acc, err}
	}()

	select {
	case r := <-ch:
		return r.acc, r.err
	case <-time.After(t.timeout):
		log.Info("timeout", "op", "list")
		go func() { <-ch }()
		return accounts.Account{}, ErrTimeout
	}
}

func (t *TimedExternalAPI) SignTransaction(ctx context.Context, args SendTxArgs, methodSelector *string) (*ethapi.SignTransactionResult, error) {
	type response struct {
		res *ethapi.SignTransactionResult
		err error
	}
	ch := make(chan response)
	go func() {
		res, err := t.next.SignTransaction(ctx, args, methodSelector)
		ch <- response{res, err}
	}()
	select {
	case r := <-ch:
		return r.res, r.err
	case <-time.After(t.timeout):
		log.Info("timeout", "op", "signTransaction")
		go func() { <-ch }()
		return nil, ErrTimeout
	}
}

func (t *TimedExternalAPI) Sign(ctx context.Context, addr common.MixedcaseAddress, data hexutil.Bytes) (hexutil.Bytes, error) {
	type response struct {
		res hexutil.Bytes
		err error
	}
	ch := make(chan response)
	go func() {
		res, err := t.next.Sign(ctx, addr, data)
		ch <- response{res, err}
	}()
	select {
	case r := <-ch:
		return r.res, r.err
	case <-time.After(t.timeout):
		log.Info("timeout", "op", "sign")
		go func() { <-ch }()
		return nil, ErrTimeout
	}
}

func (t *TimedExternalAPI) Export(ctx context.Context, addr common.Address) (json.RawMessage, error) {
	type response struct {
		res json.RawMessage
		err error
	}
	ch := make(chan response)
	go func() {
		res, err := t.next.Export(ctx, addr)
		ch <- response{res, err}
	}()
	select {
	case r := <-ch:
		return r.res, r.err
	case <-time.After(t.timeout):
		log.Info("timeout", "op", "sign")
		go func() { <-ch }()
		return json.RawMessage{}, ErrTimeout
	}
}

func (t *TimedExternalAPI) Version(ctx context.Context) (string, error) {
	return ExternalAPIVersion, nil
}
