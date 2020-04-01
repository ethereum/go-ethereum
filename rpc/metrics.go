// Copyright 2020 The go-ethereum Authors
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

package rpc

import (
	"fmt"

	"github.com/ethereum/go-ethereum/metrics"
)

var (
	jsonrpcRequestGauge    = metrics.NewRegisteredGauge("rpc/jsonrpc/count/in", nil)
	successfulRequestGauge = metrics.NewRegisteredGauge("rpc/jsonrpc/count/success", nil)
	failedReqeustGauge     = metrics.NewRegisteredGauge("rpc/jsonrpc/count/failure", nil)
	jsonrpcServingTimer    = metrics.NewRegisteredTimer("rpc/jsonrpc/duration/all", nil)
)

func newJsonrpcServingTimer(method string, valid bool) metrics.Timer {
	flag := "success"
	if !valid {
		flag = "failure"
	}
	m := fmt.Sprintf("rpc/jsonrpc/duration/%s/%s", method, flag)
	return metrics.GetOrRegisterTimer(m, nil)
}
