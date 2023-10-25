// Copyright 2023 The go-ethereum Authors
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

// Package graphql provides a GraphQL interface to Ethereum node data.
package graphql

import (
	"fmt"
)

var (
	errOnlyNumberOrHash  = invalidParamsError("only one of number or hash must be specified")
	errBlockInvariant    = invalidParamsError("block objects must be instantiated with at least one of num or hash")
	errInvalidBlockRange = invalidParamsError("invalid from and to block combination: from > to")
)

const (
	errcodeDefault       = -32600
	errcodeInvalidParams = -32602
)

type graphQLError struct {
	Code    int
	Message string
}

// Error implements the github.com/graph-gophers/graphql-go.ResolverError interface.
func (e *graphQLError) Error() string {
	return fmt.Sprintf("error [%d]: %s", e.Code, e.Message)
}

// Extensions implements the github.com/graph-gophers/graphql-go.ResolverError interface.
func (e *graphQLError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"errorCode":    e.Code,
		"errorMessage": e.Message,
	}
}

// asGraphQLError wraps an error into graphQLError
func asGraphQLError(err error) error {
	if err == nil {
		return nil
	}
	return &graphQLError{Code: errcodeDefault, Message: err.Error()}
}

func invalidParamsError(msg string) error {
	return &graphQLError{Code: errcodeInvalidParams, Message: msg}
}
