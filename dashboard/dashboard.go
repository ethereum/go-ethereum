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
//go:generate go-bindata -nometadata -o assets.go -prefix assets -nocompress -pkg dashboard assets/index.html assets/bundle.js
//go:generate sh -c "sed 's#var _bundleJs#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate sh -c "sed 's#var _indexHtml#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate gofmt -w -s assets.go

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"io"

	"github.com/elastic/gosigar"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mohae/deepcopy"
	"golang.org/x/net/websocket"
)

const (
	activeMemorySampleLimit   = 200 // Maximum number of active memory data samples
	virtualMemorySampleLimit  = 200 // Maximum number of virtual memory data samples
	networkIngressSampleLimit = 200 // Maximum number of network ingress data samples
	networkEgressSampleLimit  = 200 // Maximum number of network egress data samples
	processCPUSampleLimit     = 200 // Maximum number of process cpu data samples
	systemCPUSampleLimit      = 200 // Maximum number of system cpu data samples
	diskReadSampleLimit       = 200 // Maximum number of disk read data samples
	diskWriteSampleLimit      = 200 // Maximum number of disk write data samples
)

var nextID uint32 // Next connection id

// Dashboard contains the dashboard internals.
type Dashboard struct {
	config *Config

	listener net.Listener
	conns    map[uint32]*client // Currently live websocket connections
	history  *Message
	lock     sync.RWMutex // Lock protecting the dashboard's internals

	logdir string

	quit chan chan error // Channel used for graceful exit
	wg   sync.WaitGroup
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
				ActiveMemory:   emptyChartEntries(now, activeMemorySampleLimit, config.Refresh),
				VirtualMemory:  emptyChartEntries(now, virtualMemorySampleLimit, config.Refresh),
				NetworkIngress: emptyChartEntries(now, networkIngressSampleLimit, config.Refresh),
				NetworkEgress:  emptyChartEntries(now, networkEgressSampleLimit, config.Refresh),
				ProcessCPU:     emptyChartEntries(now, processCPUSampleLimit, config.Refresh),
				SystemCPU:      emptyChartEntries(now, systemCPUSampleLimit, config.Refresh),
				DiskRead:       emptyChartEntries(now, diskReadSampleLimit, config.Refresh),
				DiskWrite:      emptyChartEntries(now, diskWriteSampleLimit, config.Refresh),
			},
		},
		logdir: logdir,
	}
}

// emptyChartEntries returns a ChartEntry array containing limit number of empty samples.
func emptyChartEntries(t time.Time, limit int, refresh time.Duration) ChartEntries {
	ce := make(ChartEntries, limit)
	for i := 0; i < limit; i++ {
		ce[i] = &ChartEntry{
			Time: t.Add(-time.Duration(i) * refresh),
		}
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

	db.wg.Add(2)
	go db.collectData()
	go db.streamLogs()

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
	for i := 0; i < 2; i++ {
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
	id := atomic.AddUint32(&nextID, 1)
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

	db.lock.Lock()
	// Send the past data.
	client.msg <- deepcopy.Copy(db.history).(*Message)
	// Start tracking the connection and drop at connection loss.
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

// meterCollector returns a function, which retrieves a specific meter.
func meterCollector(name string) func() int64 {
	if metric := metrics.DefaultRegistry.Get(name); metric != nil {
		m := metric.(metrics.Meter)
		return func() int64 {
			return m.Count()
		}
	}
	return func() int64 {
		return 0
	}
}

// collectData collects the required data to plot on the dashboard.
func (db *Dashboard) collectData() {
	defer db.wg.Done()

	systemCPUUsage := gosigar.Cpu{}
	systemCPUUsage.Get()
	var (
		mem runtime.MemStats

		collectNetworkIngress = meterCollector("p2p/InboundTraffic")
		collectNetworkEgress  = meterCollector("p2p/OutboundTraffic")
		collectDiskRead       = meterCollector("eth/db/chaindata/disk/read")
		collectDiskWrite      = meterCollector("eth/db/chaindata/disk/write")

		prevNetworkIngress = collectNetworkIngress()
		prevNetworkEgress  = collectNetworkEgress()
		prevProcessCPUTime = getProcessCPUTime()
		prevSystemCPUUsage = systemCPUUsage
		prevDiskRead       = collectDiskRead()
		prevDiskWrite      = collectDiskWrite()

		frequency = float64(db.config.Refresh / time.Second)
		numCPU    = float64(runtime.NumCPU())
	)

	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh):
			systemCPUUsage.Get()
			var (
				curNetworkIngress = collectNetworkIngress()
				curNetworkEgress  = collectNetworkEgress()
				curProcessCPUTime = getProcessCPUTime()
				curSystemCPUUsage = systemCPUUsage
				curDiskRead       = collectDiskRead()
				curDiskWrite      = collectDiskWrite()

				deltaNetworkIngress = float64(curNetworkIngress - prevNetworkIngress)
				deltaNetworkEgress  = float64(curNetworkEgress - prevNetworkEgress)
				deltaProcessCPUTime = curProcessCPUTime - prevProcessCPUTime
				deltaSystemCPUUsage = curSystemCPUUsage.Delta(prevSystemCPUUsage)
				deltaDiskRead       = curDiskRead - prevDiskRead
				deltaDiskWrite      = curDiskWrite - prevDiskWrite
			)
			prevNetworkIngress = curNetworkIngress
			prevNetworkEgress = curNetworkEgress
			prevProcessCPUTime = curProcessCPUTime
			prevSystemCPUUsage = curSystemCPUUsage
			prevDiskRead = curDiskRead
			prevDiskWrite = curDiskWrite

			now := time.Now()

			runtime.ReadMemStats(&mem)
			activeMemory := &ChartEntry{
				Time:  now,
				Value: float64(mem.Alloc) / frequency,
			}
			virtualMemory := &ChartEntry{
				Time:  now,
				Value: float64(mem.Sys) / frequency,
			}
			networkIngress := &ChartEntry{
				Time:  now,
				Value: deltaNetworkIngress / frequency,
			}
			networkEgress := &ChartEntry{
				Time:  now,
				Value: deltaNetworkEgress / frequency,
			}
			processCPU := &ChartEntry{
				Time:  now,
				Value: deltaProcessCPUTime / frequency / numCPU * 100,
			}
			systemCPU := &ChartEntry{
				Time:  now,
				Value: float64(deltaSystemCPUUsage.Sys+deltaSystemCPUUsage.User) / frequency / numCPU,
			}
			diskRead := &ChartEntry{
				Time:  now,
				Value: float64(deltaDiskRead) / frequency,
			}
			diskWrite := &ChartEntry{
				Time:  now,
				Value: float64(deltaDiskWrite) / frequency,
			}
			sys := db.history.System
			db.lock.Lock()
			sys.ActiveMemory = append(sys.ActiveMemory[1:], activeMemory)
			sys.VirtualMemory = append(sys.VirtualMemory[1:], virtualMemory)
			sys.NetworkIngress = append(sys.NetworkIngress[1:], networkIngress)
			sys.NetworkEgress = append(sys.NetworkEgress[1:], networkEgress)
			sys.ProcessCPU = append(sys.ProcessCPU[1:], processCPU)
			sys.SystemCPU = append(sys.SystemCPU[1:], systemCPU)
			sys.DiskRead = append(sys.DiskRead[1:], diskRead)
			sys.DiskWrite = append(sys.DiskWrite[1:], diskWrite)
			db.lock.Unlock()

			db.sendToAll(&Message{
				System: &SystemMessage{
					ActiveMemory:   ChartEntries{activeMemory},
					VirtualMemory:  ChartEntries{virtualMemory},
					NetworkIngress: ChartEntries{networkIngress},
					NetworkEgress:  ChartEntries{networkEgress},
					ProcessCPU:     ChartEntries{processCPU},
					SystemCPU:      ChartEntries{systemCPU},
					DiskRead:       ChartEntries{diskRead},
					DiskWrite:      ChartEntries{diskWrite},
				},
			})
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
