// The MIT License (MIT)
//
// Copyright (c) 2016 Muhammed Thanish
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package graphql

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/graphql/internal/graphiql"
)

// GraphiQL is an in-browser IDE for exploring GraphiQL APIs.
// This handler returns GraphiQL when requested.
//
// For more information, see https://github.com/graphql/graphiql.
type GraphiQL struct{}

func respond(w http.ResponseWriter, body []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

func errorJSON(msg string) []byte {
	buf := bytes.Buffer{}
	fmt.Fprintf(&buf, `{"error": "%s"}`, msg)
	return buf.Bytes()
}

func (h GraphiQL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respond(w, errorJSON("only GET requests are supported"), http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path == "/graphql/ui/graphiql.min.css" {
		w.Header().Set("Content-Type", "text/css")
		w.Write(graphiql.Assets["graphiql.min.css"])
		return
	} else if r.URL.Path == "/graphql/ui/graphiql.min.js" {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(graphiql.Assets["graphiql.min.js"])
		return
	} else if r.URL.Path == "/graphql/ui/react.production.min.js" {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(graphiql.Assets["react.production.min.js"])
		return
	} else if r.URL.Path == "/graphql/ui/react-dom.production.min.js" {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(graphiql.Assets["react-dom.production.min.js"])
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(graphiql.Assets["index.html"])
}
