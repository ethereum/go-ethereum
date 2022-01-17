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

	"errors"
	"github.com/golang-jwt/jwt/v4"
	"strings"
	"time"
)

// customClaim implements claims.Claim.
type customClaim struct {
	// the `iat` (Issued At) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.6
	IssuedAt int64 `json:"iat,omitempty"`
}

// Valid implements claims.Claim, and checks that the iat is present and valid.
func (c customClaim) Valid() error {
	if time.Now().Unix()-5 < c.IssuedAt {
		return errors.New("token issuance (iat) is too old")
	}
	if time.Now().Unix()+5 > c.IssuedAt {
		return errors.New("token issuance (iat) is too far in the future")
	}
	return nil
}

type jwtHandler struct {
	keyFunc func(token *jwt.Token) (interface{}, error)
	next    http.Handler
}

// MakeJWTValidator creates a validator for jwt tokens.
func newJWTHandler(secret []byte, next http.Handler) http.Handler {
	return &jwtHandler{
		keyFunc: func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		},
		next: next,
	}
}

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
	t, err := jwt.ParseWithClaims(token, claims, handler.keyFunc, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		http.Error(out, err.Error(), http.StatusForbidden)
		return
	}
	if !t.Valid {
		// This should not happen, but better safe than sorry if the implementation changes.
		http.Error(out, "invalid token", http.StatusForbidden)
		return
	}
	handler.next.ServeHTTP(out, r)
}
