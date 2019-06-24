package httpu

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTPUClient is a client for dealing with HTTPU (HTTP over UDP). Its typical
// function is for HTTPMU, and particularly SSDP.
type HTTPUClient struct {
	connLock sync.Mutex // Protects use of conn.
	conn     net.PacketConn
}

// NewHTTPUClient creates a new HTTPUClient, opening up a new UDP socket for the
// purpose.
func NewHTTPUClient() (*HTTPUClient, error) {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}
	return &HTTPUClient{conn: conn}, nil
}

// NewHTTPUClientAddr creates a new HTTPUClient which will broadcast packets
// from the specified address, opening up a new UDP socket for the purpose
func NewHTTPUClientAddr(addr string) (*HTTPUClient, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, errors.New("Invalid listening address")
	}
	conn, err := net.ListenPacket("udp", ip.String()+":0")
	if err != nil {
		return nil, err
	}
	return &HTTPUClient{conn: conn}, nil
}

// Close shuts down the client. The client will no longer be useful following
// this.
func (httpu *HTTPUClient) Close() error {
	httpu.connLock.Lock()
	defer httpu.connLock.Unlock()
	return httpu.conn.Close()
}

// Do performs a request. The timeout is how long to wait for before returning
// the responses that were received. An error is only returned for failing to
// send the request. Failures in receipt simply do not add to the resulting
// responses.
//
// Note that at present only one concurrent connection will happen per
// HTTPUClient.
func (httpu *HTTPUClient) Do(req *http.Request, timeout time.Duration, numSends int) ([]*http.Response, error) {
	httpu.connLock.Lock()
	defer httpu.connLock.Unlock()

	// Create the request. This is a subset of what http.Request.Write does
	// deliberately to avoid creating extra fields which may confuse some
	// devices.
	var requestBuf bytes.Buffer
	method := req.Method
	if method == "" {
		method = "GET"
	}
	if _, err := fmt.Fprintf(&requestBuf, "%s %s HTTP/1.1\r\n", method, req.URL.RequestURI()); err != nil {
		return nil, err
	}
	if err := req.Header.Write(&requestBuf); err != nil {
		return nil, err
	}
	if _, err := requestBuf.Write([]byte{'\r', '\n'}); err != nil {
		return nil, err
	}

	destAddr, err := net.ResolveUDPAddr("udp", req.Host)
	if err != nil {
		return nil, err
	}
	if err = httpu.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	// Send request.
	for i := 0; i < numSends; i++ {
		if n, err := httpu.conn.WriteTo(requestBuf.Bytes(), destAddr); err != nil {
			return nil, err
		} else if n < len(requestBuf.Bytes()) {
			return nil, fmt.Errorf("httpu: wrote %d bytes rather than full %d in request",
				n, len(requestBuf.Bytes()))
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Await responses until timeout.
	var responses []*http.Response
	responseBytes := make([]byte, 2048)
	for {
		// 2048 bytes should be sufficient for most networks.
		n, _, err := httpu.conn.ReadFrom(responseBytes)
		if err != nil {
			if err, ok := err.(net.Error); ok {
				if err.Timeout() {
					break
				}
				if err.Temporary() {
					// Sleep in case this is a persistent error to avoid pegging CPU until deadline.
					time.Sleep(10 * time.Millisecond)
					continue
				}
			}
			return nil, err
		}

		// Parse response.
		response, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(responseBytes[:n])), req)
		if err != nil {
			log.Printf("httpu: error while parsing response: %v", err)
			continue
		}

		responses = append(responses, response)
	}

	// Timeout reached - return discovered responses.
	return responses, nil
}
