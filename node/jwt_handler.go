// Copyright 2022 The go-ethereum Authors
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

package node

import (
	"net/http"

	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/golang-jwt/jwt/v4"
	"strings"
	"time"
)

type jwtHandler struct {
	keyFunc func(token *jwt.Token) (interface{}, error)
	next    http.Handler
}

// newJWTHandler creates a http.Handler with jwt authentication support.
func newJWTHandler(secret []byte, next http.Handler) http.Handler {
	return &jwtHandler{
		keyFunc: func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		},
		next: next,
	}
}

// customClaim is basically a standard RegisteredClaim, but we override the
// Valid method to be more lax in allowing some time skew.
type customClaim jwt.RegisteredClaims

// Valid implements jwt.Claim. This method only validates the (optional) expiry-time.
func (c customClaim) Valid() error {
	now := jwt.TimeFunc()
	rc := jwt.RegisteredClaims(c)
	if !rc.VerifyExpiresAt(now, false) { // optional
		return fmt.Errorf("token is expired")
	}
	if c.IssuedAt == nil {
		return fmt.Errorf("missing issued-at")
	}
	if time.Since(c.IssuedAt.Time) > 5*time.Second {
		return fmt.Errorf("stale token")
	}
	if time.Until(c.IssuedAt.Time) > 5*time.Second {
		return fmt.Errorf("future token")
	}
	return nil
}

// ServeHTTP implements http.Handler
func (handler *jwtHandler) ServeHTTP(out http.ResponseWriter, r *http.Request) {
	var token string
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if len(token) == 0 {
		http.Error(out, "missing token", http.StatusForbidden)
		return
	}
	var claims customClaim
	t, err := jwt.ParseWithClaims(token, &claims, handler.keyFunc, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		log.Info("Token parsing failed", "err", err)
		http.Error(out, err.Error(), http.StatusForbidden)
		return
	}
	if !t.Valid {
		http.Error(out, "invalid token", http.StatusForbidden)
		return
	}
	handler.next.ServeHTTP(out, r)
}
