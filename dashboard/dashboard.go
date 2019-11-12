// Copyright 2017 The go-ethereum Authors
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

package dashboard

//go:generate yarn --cwd ./assets install
//go:generate yarn --cwd ./assets build
//go:generate yarn --cwd ./assets js-beautify -f bundle.js.map -r -w 1
//go:generate go-bindata -nometadata -o assets.go -prefix assets -nocompress -pkg dashboard assets/index.html assets/bundle.js assets/bundle.js.map
//go:generate sh -c "sed 's#var _bundleJs#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate sh -c "sed 's#var _bundleJsMap#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate sh -c "sed 's#var _indexHtml#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate gofmt -w -s assets.go

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"io"

	"github.com/maticnetwork/bor/log"
	"github.com/maticnetwork/bor/p2p"
	"github.com/maticnetwork/bor/params"
	"github.com/maticnetwork/bor/rpc"
	"github.com/mohae/deepcopy"
	"golang.org/x/net/websocket"
)

const (
	sampleLimit = 200 // Maximum number of data samples
)

// Dashboard contains the dashboard internals.
type Dashboard struct {
	config *Config // Configuration values for the dashboard

	listener   net.Listener       // Network listener listening for dashboard clients
	conns      map[uint32]*client // Currently live websocket connections
	nextConnID uint32             // Next connection id

	history *Message // Stored historical data

	lock     sync.Mutex   // Lock protecting the dashboard's internals
	sysLock  sync.RWMutex // Lock protecting the stored system data
	peerLock sync.RWMutex // Lock protecting the stored peer data
	logLock  sync.RWMutex // Lock protecting the stored log data

	geodb  *geoDB // geoip database instance for IP to geographical information conversions
	logdir string // Directory containing the log files

	quit chan chan error // Channel used for graceful exit
	wg   sync.WaitGroup  // Wait group used to close the data collector threads
}

// client represents active websocket connection with a remote browser.
type client struct {
	conn   *websocket.Conn // Particular live websocket connection
	msg    chan *Message   // Message queue for the update messages
	logger log.Logger      // Logger for the particular live websocket connection
}

// New creates a new dashboard instance with the given configuration.
func New(config *Config, commit string, logdir string) *Dashboard {
	now := time.Now()
	versionMeta := ""
	if len(params.VersionMeta) > 0 {
		versionMeta = fmt.Sprintf(" (%s)", params.VersionMeta)
	}
	return &Dashboard{
		conns:  make(map[uint32]*client),
		config: config,
		quit:   make(chan chan error),
		history: &Message{
			General: &GeneralMessage{
				Commit:  commit,
				Version: fmt.Sprintf("v%d.%d.%d%s", params.VersionMajor, params.VersionMinor, params.VersionPatch, versionMeta),
			},
			System: &SystemMessage{
				ActiveMemory:   emptyChartEntries(now, sampleLimit),
				VirtualMemory:  emptyChartEntries(now, sampleLimit),
				NetworkIngress: emptyChartEntries(now, sampleLimit),
				NetworkEgress:  emptyChartEntries(now, sampleLimit),
				ProcessCPU:     emptyChartEntries(now, sampleLimit),
				SystemCPU:      emptyChartEntries(now, sampleLimit),
				DiskRead:       emptyChartEntries(now, sampleLimit),
				DiskWrite:      emptyChartEntries(now, sampleLimit),
			},
		},
		logdir: logdir,
	}
}

// emptyChartEntries returns a ChartEntry array containing limit number of empty samples.
func emptyChartEntries(t time.Time, limit int) ChartEntries {
	ce := make(ChartEntries, limit)
	for i := 0; i < limit; i++ {
		ce[i] = new(ChartEntry)
	}
	return ce
}

// Protocols implements the node.Service interface.
func (db *Dashboard) Protocols() []p2p.Protocol { return nil }

// APIs implements the node.Service interface.
func (db *Dashboard) APIs() []rpc.API { return nil }

// Start starts the data collection thread and the listening server of the dashboard.
// Implements the node.Service interface.
func (db *Dashboard) Start(server *p2p.Server) error {
	log.Info("Starting dashboard")

	db.wg.Add(3)
	go db.collectSystemData()
	go db.streamLogs()
	go db.collectPeerData()

	http.HandleFunc("/", db.webHandler)
	http.Handle("/api", websocket.Handler(db.apiHandler))

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", db.config.Host, db.config.Port))
	if err != nil {
		return err
	}
	db.listener = listener

	go http.Serve(listener, nil)

	return nil
}

// Stop stops the data collection thread and the connection listener of the dashboard.
// Implements the node.Service interface.
func (db *Dashboard) Stop() error {
	// Close the connection listener.
	var errs []error
	if err := db.listener.Close(); err != nil {
		errs = append(errs, err)
	}
	// Close the collectors.
	errc := make(chan error, 1)
	for i := 0; i < 3; i++ {
		db.quit <- errc
		if err := <-errc; err != nil {
			errs = append(errs, err)
		}
	}
	// Close the connections.
	db.lock.Lock()
	for _, c := range db.conns {
		if err := c.conn.Close(); err != nil {
			c.logger.Warn("Failed to close connection", "err", err)
		}
	}
	db.lock.Unlock()

	// Wait until every goroutine terminates.
	db.wg.Wait()
	log.Info("Dashboard stopped")

	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("%v", errs)
	}

	return err
}

// webHandler handles all non-api requests, simply flattening and returning the dashboard website.
func (db *Dashboard) webHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("Request", "URL", r.URL)

	path := r.URL.String()
	if path == "/" {
		path = "/index.html"
	}
	blob, err := Asset(path[1:])
	if err != nil {
		log.Warn("Failed to load the asset", "path", path, "err", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write(blob)
}

// apiHandler handles requests for the dashboard.
func (db *Dashboard) apiHandler(conn *websocket.Conn) {
	id := atomic.AddUint32(&db.nextConnID, 1)
	client := &client{
		conn:   conn,
		msg:    make(chan *Message, 128),
		logger: log.New("id", id),
	}
	done := make(chan struct{})

	// Start listening for messages to send.
	db.wg.Add(1)
	go func() {
		defer db.wg.Done()

		for {
			select {
			case <-done:
				return
			case msg := <-client.msg:
				if err := websocket.JSON.Send(client.conn, msg); err != nil {
					client.logger.Warn("Failed to send the message", "msg", msg, "err", err)
					client.conn.Close()
					return
				}
			}
		}
	}()

	// Send the past data.
	db.sysLock.RLock()
	db.peerLock.RLock()
	db.logLock.RLock()

	h := deepcopy.Copy(db.history).(*Message)

	db.sysLock.RUnlock()
	db.peerLock.RUnlock()
	db.logLock.RUnlock()

	client.msg <- h

	// Start tracking the connection and drop at connection loss.
	db.lock.Lock()
	db.conns[id] = client
	db.lock.Unlock()
	defer func() {
		db.lock.Lock()
		delete(db.conns, id)
		db.lock.Unlock()
	}()
	for {
		r := new(Request)
		if err := websocket.JSON.Receive(conn, r); err != nil {
			if err != io.EOF {
				client.logger.Warn("Failed to receive request", "err", err)
			}
			close(done)
			return
		}
		if r.Logs != nil {
			db.handleLogRequest(r.Logs, client)
		}
	}
}

// sendToAll sends the given message to the active dashboards.
func (db *Dashboard) sendToAll(msg *Message) {
	db.lock.Lock()
	for _, c := range db.conns {
		select {
		case c.msg <- msg:
		default:
			c.conn.Close()
		}
	}
	db.lock.Unlock()
}
