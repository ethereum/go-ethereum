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
	w.Header().Set("Content-Type", "text/html")
	w.Write(graphiql)
}

var graphiql = []byte(`
<!DOCTYPE html>
<html>
	<head>
		<link
                rel="icon"
                type="image/png"
                href="data:image/png;base64,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"
        />
        <link
                rel="stylesheet"
                href="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.13.0/graphiql.css"
                integrity="sha384-Qua2xoKBxcHOg1ivsKWo98zSI5KD/UuBpzMIg8coBd4/jGYoxeozCYFI9fesatT0"
                crossorigin="anonymous"
        />
        <script
                src="https://cdnjs.cloudflare.com/ajax/libs/fetch/3.0.0/fetch.min.js"
                integrity="sha384-5B8/4F9AQqp/HCHReGLSOWbyAOwnJsPrvx6C0+VPUr44Olzi99zYT1xbVh+ZanQJ"
                crossorigin="anonymous"
        ></script>
        <script
                src="https://cdnjs.cloudflare.com/ajax/libs/react/16.8.5/umd/react.production.min.js"
                integrity="sha384-dOCiLz3nZfHiJj//EWxjwSKSC6Z1IJtyIEK/b/xlHVNdVLXDYSesoxiZb94bbuGE"
                crossorigin="anonymous"
        ></script>
        <script
                src="https://cdnjs.cloudflare.com/ajax/libs/react-dom/16.8.5/umd/react-dom.production.min.js"
                integrity="sha384-QI+ql5f+khgo3mMdCktQ3E7wUKbIpuQo8S5rA/3i1jg2rMsloCNyiZclI7sFQUGN"
                crossorigin="anonymous"
        ></script>
        <script
                src="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.13.0/graphiql.min.js"
                integrity="sha384-roSmzNmO4zJK9X4lwggDi4/oVy+9V4nlS1+MN8Taj7tftJy1GvMWyAhTNXdC/fFR"
                crossorigin="anonymous"
        ></script>
	</head>
	<body style="width: 100%; height: 100%; margin: 0; overflow: hidden;">
		<div id="graphiql" style="height: 100vh;">Loading...</div>
		<script>
			function fetchGQL(params) {
				return fetch("/graphql", {
					method: "post",
					body: JSON.stringify(params),
					credentials: "include",
				}).then(function (resp) {
					return resp.text();
				}).then(function (body) {
					try {
						return JSON.parse(body);
					} catch (error) {
						return body;
					}
				});
			}
			ReactDOM.render(
				React.createElement(GraphiQL, {fetcher: fetchGQL}),
				document.getElementById("graphiql")
			)
		</script>
	</body>
</html>
`)
