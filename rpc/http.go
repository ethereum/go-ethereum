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

package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/fatih/set.v0"
)

const (
	httpReadDeadLine = 60 * time.Second // wait max httpReadDeadeline for next request
)

// httpMessageStream is the glue between a HTTP connection which is message based
// and the RPC codecs that expect json requests to be read from a stream. It will
// parse HTTP messages and offer the bodies of these requests as a stream through
// the Read method. This will require full control of the connection and thus need
// a "hijacked" HTTP connection.
type httpMessageStream struct {
	conn             net.Conn          // TCP connection
	rw               *bufio.ReadWriter // buffered where HTTP requests/responses are read/written from/to
	currentReq       *http.Request     // pending request, codec can pass in a too small buffer for a single read we need to keep track of the current requests if it was not read at once
	payloadBytesRead int64             // number of bytes which are read from the current request
	allowedOrigins   *set.Set          // allowed CORS domains
	origin           string            // origin of this connection/request
}

// NewHttpMessageStream will create a new http message stream parser that can be
// used by the codes in the RPC package. It will take full control of the given
// connection and thus needs to be hijacked. It will read and write HTTP messages
// from the passed rwbuf. The allowed origins are the RPC CORS domains the user has supplied.
func NewHTTPMessageStream(c net.Conn, rwbuf *bufio.ReadWriter, initialReq *http.Request, allowdOrigins []string) *httpMessageStream {
	r := &httpMessageStream{conn: c, rw: rwbuf, currentReq: initialReq, allowedOrigins: set.New()}
	for _, origin := range allowdOrigins {
		r.allowedOrigins.Add(origin)
	}
	return r
}

// handleOptionsRequest handles the HTTP preflight requests (OPTIONS) that browsers
// make to enforce CORS rules. Only the POST method is allowed and the origin must
// be on the rpccorsdomain list the user has specified.
func (h *httpMessageStream) handleOptionsRequest(req *http.Request) error {
	headers := req.Header

	if !strings.EqualFold(req.Method, "OPTIONS") {
		return fmt.Errorf("preflight aborted: %s!=OPTIONS", req.Method)
	}

	origin := headers.Get("Origin")
	if origin == "" {
		return fmt.Errorf("preflight aborted: empty origin")
	}

	responseHeaders := make(http.Header)
	responseHeaders.Set("Access-Control-Allow-Methods", "POST")
	if h.allowedOrigins.Has(origin) || h.allowedOrigins.Has("*") {
		responseHeaders.Set("Access-Control-Allow-Origin", origin)
	} else {
		glog.V(logger.Info).Infof("origin '%s' not allowed", origin)
	}
	responseHeaders.Set("Access-Control-Allow-Headers", "Content-Type")
	responseHeaders.Set("Date", string(httpTimestamp(time.Now())))
	responseHeaders.Set("Content-Type", "text/plain; charset=utf-8")
	responseHeaders.Set("Content-Length", "0")
	responseHeaders.Set("Vary", "Origin")

	defer h.rw.Flush()

	if _, err := h.rw.WriteString("HTTP/1.1 200 OK\r\n"); err != nil {
		glog.V(logger.Error).Infof("unable to write OPTIONS response: %v\n", err)
		return err
	}
	if err := responseHeaders.Write(h.rw); err != nil {
		glog.V(logger.Error).Infof("unable to write OPTIONS headers: %v\n", err)
	}
	if _, err := h.rw.WriteString("\r\n"); err != nil {
		glog.V(logger.Error).Infof("unable to write OPTIONS response: %v\n", err)
	}

	return nil
}

// Read will read incoming HTTP requests and reads the body data from these requests
// as an endless stream of data.
func (h *httpMessageStream) Read(buf []byte) (n int, err error) {
	h.conn.SetReadDeadline(time.Now().Add(httpReadDeadLine))
	for {
		// if the last request was read completely try to read the next request
		if h.currentReq == nil {
			if h.currentReq, err = http.ReadRequest(bufio.NewReader(h.rw)); err != nil {
				return 0, err
			}
		}

		// The "options" method is http specific and not interested for the RPC server.
		// Handle it internally and wait for the next request.
		if strings.EqualFold(h.currentReq.Method, "OPTIONS") {
			if err = h.handleOptionsRequest(h.currentReq); err != nil {
				glog.V(logger.Info).Infof("RPC/HTTP OPTIONS error: %v\n", err)
				h.currentReq = nil
				return 0, err
			}

			// processed valid request -> reset deadline
			h.conn.SetReadDeadline(time.Now().Add(httpReadDeadLine))
			h.currentReq = nil
			continue
		}

		if strings.EqualFold(h.currentReq.Method, "GET") || strings.EqualFold(h.currentReq.Method, "POST") {
			n, err := h.currentReq.Body.Read(buf)
			h.payloadBytesRead += int64(n)

			// entire payload read, read new request next time
			if err == io.EOF || h.payloadBytesRead >= h.currentReq.ContentLength {
				h.origin = h.currentReq.Header.Get("origin")
				h.payloadBytesRead = 0
				h.currentReq.Body.Close()
				h.currentReq = nil
				err = nil // io.EOF is not an error
			} else if err != nil {
				// unable to read body
				h.currentReq.Body.Close()
				h.currentReq = nil
				h.payloadBytesRead = 0
			}
			// partial read of body
			return n, err
		}
		return 0, fmt.Errorf("unsupported HTTP method '%s'", h.currentReq.Method)
	}
}

// Write will create a HTTP response with the given payload and send it to the peer.
func (h *httpMessageStream) Write(payload []byte) (int, error) {
	defer h.rw.Flush()

	responseHeaders := make(http.Header)
	responseHeaders.Set("Content-Type", "application/json")
	responseHeaders.Set("Content-Length", strconv.Itoa(len(payload)))
	if h.origin != "" {
		responseHeaders.Set("Access-Control-Allow-Origin", h.origin)
	}

	h.rw.WriteString("HTTP/1.1 200 OK\r\n")
	responseHeaders.Write(h.rw)
	h.rw.WriteString("\r\n")

	return h.rw.Write(payload)
}

// Close will close the underlying TCP connection this instance has taken ownership over.
func (h *httpMessageStream) Close() error {
	h.rw.Flush()
	return h.conn.Close()
}

// TimeFormat is the time format to use with time.Parse and time.Time.Format when
// parsing or generating times in HTTP headers. It is like time.RFC1123 but hard
// codes GMT as the time zone.
const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// httpTimestamp formats the given t as specified in RFC1123.
func httpTimestamp(t time.Time) []byte {
	const days = "SunMonTueWedThuFriSat"
	const months = "JanFebMarAprMayJunJulAugSepOctNovDec"

	b := make([]byte, 0)
	t = t.UTC()
	yy, mm, dd := t.Date()
	hh, mn, ss := t.Clock()
	day := days[3*t.Weekday():]
	mon := months[3*(mm-1):]

	return append(b,
		day[0], day[1], day[2], ',', ' ',
		byte('0'+dd/10), byte('0'+dd%10), ' ',
		mon[0], mon[1], mon[2], ' ',
		byte('0'+yy/1000), byte('0'+(yy/100)%10), byte('0'+(yy/10)%10), byte('0'+yy%10), ' ',
		byte('0'+hh/10), byte('0'+hh%10), ':',
		byte('0'+mn/10), byte('0'+mn%10), ':',
		byte('0'+ss/10), byte('0'+ss%10), ' ',
		'G', 'M', 'T')
}

// httpConnHijacker is a http.Handler implementation that will hijack the HTTP
// connection,  wraps it in a HttpMessageStream that is then wrapped in a JSON
// codec which will be served on the rpcServer.
type httpConnHijacker struct {
	corsdomains []string
	rpcServer   *Server
}

// ServeHTTP will hijack the connection, wraps the captured connection in a
// HttpMessageStream which is then used as codec.
func (h *httpConnHijacker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	conn, rwbuf, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httpRequestStream := NewHTTPMessageStream(conn, rwbuf, req, h.corsdomains)

	codec := NewJSONCodec(httpRequestStream)
	go h.rpcServer.ServeCodec(codec)
}

// NewHTTPServer creates a new HTTP RPC server around an API provider.
func NewHTTPServer(cors string, handler *Server) *http.Server {
	return &http.Server{
		Handler: &httpConnHijacker{
			corsdomains: strings.Split(cors, ","),
			rpcServer:   handler,
		},
	}
}

// httpClient connects to a geth RPC server over HTTP.
type httpClient struct {
	endpoint *url.URL // HTTP-RPC server endpoint
	lastRes  []byte   // HTTP requests are synchronous, store last response
}

// NewHTTPClient create a new RPC clients that connection to a geth RPC server
// over HTTP.
func NewHTTPClient(endpoint string) (Client, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &httpClient{endpoint: url}, nil
}

// Send will serialize the given msg to JSON and sends it to the RPC server.
// Since HTTP is synchronous the response is stored until Recv is called.
func (client *httpClient) Send(msg interface{}) error {
	var body []byte
	var err error

	client.lastRes = nil

	if body, err = json.Marshal(msg); err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", client.endpoint.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		client.lastRes, err = ioutil.ReadAll(resp.Body)
		return err
	}

	return fmt.Errorf("unable to handle request")
}

// Recv will try to deserialize the last received response into the given msg.
func (client *httpClient) Recv(msg interface{}) error {
	return json.Unmarshal(client.lastRes, &msg)
}

// Close is not necessary for httpClient
func (client *httpClient) Close() {
}

// SupportedModules will return the collection of offered RPC modules.
func (client *httpClient) SupportedModules() (map[string]string, error) {
	return SupportedModules(client)
}
