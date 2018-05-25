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

	"encoding/json"
	"github.com/elastic/gosigar"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/fsnotify/fsnotify"
	"github.com/mohae/deepcopy"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
func New(config *Config, commit string, logdir string) (*Dashboard, error) {
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
			Logs: &LogsMessage{
				Stream: true,
				End:    false,
				Chunk:  json.RawMessage("[]"),
			},
		},
		logdir: logdir,
	}, nil
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

// Protocols is a meaningless implementation of node.Service.
func (db *Dashboard) Protocols() []p2p.Protocol { return nil }

// APIs is a meaningless implementation of node.Service.
func (db *Dashboard) APIs() []rpc.API { return nil }

// Start implements node.Service, starting the data collection thread and the listening server of the dashboard.
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

// Stop implements node.Service, stopping the data collection thread and the connection listener of the dashboard.
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
		var r Request
		err := websocket.JSON.Receive(conn, &r)
		if err != nil {
			client.logger.Warn("Failed to receive request", "err", err)
			close(done)
			return
		}
		if r.Logs != nil {
			db.handleLogRequest(r.Logs, client) // TODO (kurkomisi): concurrent function call?
		}
	}
}

func validateLogFile(path string) ([]byte, bool) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		log.Warn("Failed to open file", "path", path, "err", err)
		return nil, false
	}
	defer f.Close()
	var buf []byte
	if buf, err = ioutil.ReadAll(f); err != nil {
		log.Warn("Failed to read file", "path", path, "err", err)
		return nil, false
	}
	end := -1
	for j := 0; j < len(buf); j++ {
		if buf[j] == '\n' {
			buf[j] = ','
			end = j
		}
	}
	if end < 0 {
		return nil, false
	}

	return buf[:end], true
}

// handleLogRequest searches for the log file specified by the timestamp of the request, creates a JSON array out of it
// and sends it to the requesting client.
func (db *Dashboard) handleLogRequest(r *LogsRequest, c *client) {
	files, err := ioutil.ReadDir(db.logdir)
	if err != nil {
		log.Warn("Failed to open logdir", "logdir", db.logdir, "err", err)
		return
	}
	re := regexp.MustCompile(".log$")
	fileNames := make([]string, len(files))
	n := 0
	for _, f := range files {
		if f.Mode().IsRegular() && re.Match([]byte(f.Name())) {
			fileNames[n] = f.Name()
			n++
		}
	}
	n-- // The last file is handled by the stream handler in order to avoid log duplication on the client side.
	if n < 1 {
		log.Warn("There isn't any old log file in the logdir", "path", db.logdir)
		return
	}
	timestamp := fmt.Sprintf("%s.log", strings.Replace(r.Time.Format("060102150405.00"), ".", "", 1))

	i := sort.Search(n, func(i int) bool {
		return fileNames[i] >= timestamp // Returns the smallest index such as fileNames[i] >= timestamp.
	})
	ok := false
	var buf json.RawMessage
	if r.Past {
		if i >= n {
			i = n - 1
		}
		for i >= 0 && fileNames[i] >= timestamp {
			i--
		}
		for i >= 0 && !ok {
			buf, ok = validateLogFile(filepath.Join(db.logdir, fileNames[i]))
			i--
		}
	} else {
		for i < n && fileNames[i] <= timestamp {
			i++
		}
		for i < n && !ok {
			buf, ok = validateLogFile(filepath.Join(db.logdir, fileNames[i]))
			i++
		}
	}
	if buf == nil {
		buf = json.RawMessage{}
	}
	b := make(json.RawMessage, len(buf)+2)
	b[0] = '['
	copy(b[1:], buf)
	b[len(buf)+1] = ']'

	db.lock.Lock()
	c.msg <- &Message{
		Logs: &LogsMessage{
			Stream: false,
			Past:   r.Past,
			End:    !ok,
			Chunk:  b,
		},
	}
	db.lock.Unlock()
}

// metricCollector returns a function, which retrieves a specific metric.
func metricCollector(name string) func() int64 {
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

		collectNetworkIngress = metricCollector("p2p/InboundTraffic")
		collectNetworkEgress  = metricCollector("p2p/OutboundTraffic")
		collectDiskRead       = metricCollector("eth/db/chaindata/disk/read")
		collectDiskWrite      = metricCollector("eth/db/chaindata/disk/write")

		prevNetworkIngress = collectNetworkIngress()
		prevNetworkEgress  = collectNetworkEgress()
		prevProcessCPUTime = getProcessCPUTime()
		prevSystemCPUUsage = systemCPUUsage
		prevDiskRead       = collectDiskRead()
		prevDiskWrite      = collectDiskWrite()

		frequency = float64(db.config.Refresh / time.Second)
		numCPU    = float64(runtime.NumCPU())

		sys = db.history.System
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
			sys.ActiveMemory = append(sys.ActiveMemory[1:], activeMemory)
			sys.VirtualMemory = append(sys.VirtualMemory[1:], virtualMemory)
			sys.NetworkIngress = append(sys.NetworkIngress[1:], networkIngress)
			sys.NetworkEgress = append(sys.NetworkEgress[1:], networkEgress)
			sys.ProcessCPU = append(sys.ProcessCPU[1:], processCPU)
			sys.SystemCPU = append(sys.SystemCPU[1:], systemCPU)
			sys.DiskRead = append(sys.DiskRead[1:], diskRead)
			sys.DiskWrite = append(sys.DiskRead[1:], diskWrite)

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

// streamLogs watches the file system, and when the logger writes the new log records into the files, picks them up,
// then makes JSON array out of them and sends them to the clients.
// This could be embedded into collectData, but they shouldn't depend on each other, and also cleaner this way.
func (db *Dashboard) streamLogs() {
	defer db.wg.Done()

	files, err := ioutil.ReadDir(db.logdir)
	if err != nil {
		log.Warn("Failed to open logdir", "logdir", db.logdir, "err", err)
		return
	}
	var (
		opened *os.File // File descriptor for the opened active log file.
		buf    []byte   // Contains the recently written log chunks, which are not sent to the clients yet.
	)

	// The log records are always written into the last file in alphabetical order, because of the timestamp.
	re := regexp.MustCompile(".log$")
	var i int
	for i = len(files) - 1; i >= 0 && (!files[i].Mode().IsRegular() || !re.Match([]byte(files[i].Name()))); i-- {
	}
	if i >= 0 {
		if opened, err = os.OpenFile(filepath.Join(db.logdir, files[i].Name()), os.O_RDONLY, 0644); err != nil {
			log.Warn("Failed to open file", "name", files[i].Name(), "err", err)
			return
		}
		if buf, err = ioutil.ReadAll(opened); err != nil {
			log.Warn("Failed to read file", "name", opened.Name(), "err", err)
			return
		}
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Warn("Failed to create fs watcher", "err", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(db.logdir)
	if err != nil {
		log.Warn("Failed to add logdir to fs watcher", "logdir", db.logdir, "err", err)
		return
	}
	defer opened.Close() // Close the lastly opened file.
	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	newFile := false
	for {
		select {
		case event := <-watcher.Events:
			// If new log file was created.
			if event.Op&fsnotify.Create != 0 && re.Match([]byte(event.Name)) {
				if opened != nil {
					// The new log file's timestamp is always greater, since it is created of the actual time.
					if opened.Name() >= event.Name {
						break
					}
					// Read the rest of the previously opened file.
					chunk, err := ioutil.ReadAll(opened)
					if err != nil {
						log.Warn("Failed to read file", "name", opened.Name(), "err", err)
						return
					}
					b := make([]byte, len(buf)+len(chunk))
					copy(b, buf)
					copy(b[len(buf):], chunk)
					buf = b
					opened.Close()
				}
				last := -1
				for i := 0; i < len(buf); i++ {
					if buf[i] == '\n' {
						buf[i] = ','
						last = i
					}
				}
				if last >= 0 {
					msg := make([]byte, last+2)
					msg[0] = '['
					copy(msg[1:], buf[:last])
					msg[last+1] = ']'

					db.sendToAll(&Message{
						Logs: &LogsMessage{
							Stream: true,
							End:    false,
							Chunk:  msg,
						},
					})
					db.lock.Lock()
					db.history.Logs.Chunk = json.RawMessage("[]")
					db.lock.Unlock()
				}
				buf = buf[:0]
				newFile = true
				if opened, err = os.OpenFile(event.Name, os.O_RDONLY, 0644); err != nil {
					log.Warn("Failed to open file", "name", event.Name, "err", err)
					return
				}
			}
		case err := <-watcher.Errors:
			if err != nil {
				log.Warn("Fs watcher error", "err", err)
			}
			return
		case errc := <-db.quit:
			errc <- nil
			return
		// Send log updates to the client.
		case <-ticker.C:
			if opened == nil {
				break
			}

			// Read the new logs created since the last read.
			chunk, err := ioutil.ReadAll(opened)
			if err != nil {
				log.Warn("Failed to read file", "name", opened.Name(), "err", err)
				return
			}
			b := make([]byte, len(buf)+len(chunk))
			copy(b, buf)
			copy(b[len(buf):], chunk)
			last := -1
			for i := 0; i < len(b); i++ {
				if b[i] == '\n' {
					b[i] = ','
					last = i
				}
			}
			if last < 0 {
				break
			}
			// Clear the valid/sent part of the buffer.
			buf = b[last+1:]

			msg := make([]byte, last+2)
			msg[0] = '['
			copy(msg[1:], b[:last])
			msg[last+1] = ']'

			db.sendToAll(&Message{
				Logs: &LogsMessage{
					Stream: true,
					End:    newFile,
					Chunk:  msg,
				},
			})
			newFile = false

			db.lock.Lock()
			if len(db.history.Logs.Chunk) == 2 {
				db.history.Logs.Chunk = msg
			} else {
				b = make([]byte, len(db.history.Logs.Chunk)+len(msg)-1)
				copy(b, db.history.Logs.Chunk)
				b[len(db.history.Logs.Chunk)-1] = ','
				copy(b[len(db.history.Logs.Chunk):], msg[1:])
				db.history.Logs.Chunk = b
			}
			db.lock.Unlock()
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
