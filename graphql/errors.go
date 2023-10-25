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
	"errors"
	"fmt"
)

var (
	errBlockInvariant    = errors.New("block objects must be instantiated with at least one of num or hash")
	errInvalidBlockRange = errors.New("invalid from and to block combination: from > to")
)

type graphQLError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the github.com/graph-gophers/graphql-go.ResolverError interface.
func (e *graphQLError) Error() string {
	return fmt.Sprintf("error [%s]: %s", e.Code, e.Message)
}

// Extensions implements the github.com/graph-gophers/graphql-go.ResolverError interface.
func (e *graphQLError) Extensions() map[string]interface{} {
	return map[string]interface{}{
		"code":    e.Code,
		"message": e.Message,
	}
}

// asGraphQLError wraps an error into graphQLError
func asGraphQLError(err error) error {
	if err == nil {
		return nil
	}
	return &graphQLError{Code: "-32602", Message: err.Error()}
}
