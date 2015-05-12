/*
A simple http server interface to Swarm
*/
package bzz

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

const (
	port         = ":8500"
	manifestType = "application/bzz-manifest+json"
)

var (
	protocolMatcher = regexp.MustCompile("^/bzz:")
	uriMatcher      = regexp.MustCompile("^/raw/[0-9A-Fa-f]{64}(?:/[a-z]+/[-+0-9a-z]+)?$")
	manifestMatcher = regexp.MustCompile("^/[0-9A-Fa-f]{64}")
	hashMatcher     = regexp.MustCompile("^[0-9A-Fa-f]{64}$")
)

type sequentialReader struct {
	reader io.Reader
	pos    int64
	ahead  map[int64](chan bool)
	lock   sync.Mutex
}

type manifest struct {
	Entries []manifestEntry
}

type manifestEntry struct {
	Path        string
	Hash        string
	ContentType string
	Status      int16
}

func (self *sequentialReader) ReadAt(target []byte, off int64) (n int, err error) {
	self.lock.Lock()
	// assert self.pos <= off
	if self.pos > off {
		dpaLogger.Errorf("Swarm: non-sequential read attempted from sequentialReader; %d > %d",
			self.pos, off)
		panic("Non-sequential read attempt")
	}
	if self.pos != off {
		dpaLogger.Debugf("Swarm: deferred read in POST at position %d, offset %d.",
			self.pos, off)
		wait := make(chan bool)
		self.ahead[off] = wait
		self.lock.Unlock()
		if <-wait {
			// failed read behind
			n = 0
			err = io.ErrUnexpectedEOF
			return
		}
		self.lock.Lock()
	}
	localPos := 0
	for localPos < len(target) {
		n, err = self.reader.Read(target[localPos:])
		localPos += n
		dpaLogger.Debugf("Swarm: Read %d bytes into buffer size %d from POST, error %v.",
			n, len(target), err)
		if err != nil {
			dpaLogger.Debugf("Swarm: POST stream's reading terminated with %v.", err)
			for i := range self.ahead {
				self.ahead[i] <- true
				delete(self.ahead, i)
			}
			self.lock.Unlock()
			return localPos, err
		}
		self.pos += int64(n)
	}
	wait := self.ahead[self.pos]
	if wait != nil {
		dpaLogger.Debugf("Swarm: deferred read in POST at position %d triggered.",
			self.pos)
		delete(self.ahead, self.pos)
		close(wait)
	}
	self.lock.Unlock()
	return localPos, err
}

func handler(w http.ResponseWriter, r *http.Request, dpa *DPA) {
	uri := protocolMatcher.ReplaceAllString(r.RequestURI, "")
	switch {
	case r.Method == "POST":
		if uri == "/raw" {
			dpaLogger.Debugf("Swarm: POST request received.")
			key, err := dpa.Store(io.NewSectionReader(&sequentialReader{
				reader: r.Body,
				ahead:  make(map[int64]chan bool),
			}, 0, r.ContentLength), nil)
			if err == nil {
				fmt.Fprintf(w, "%064x", key)
				dpaLogger.Debugf("Swarm: Object %064x stored", key)
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
		} else {
			http.Error(w, "No POST to "+uri+" allowed.", http.StatusBadRequest)
		}
	case r.Method == "GET":
		if uriMatcher.MatchString(uri) {
			dpaLogger.Debugf("Swarm: Raw GET request %s received", uri)
			name := uri[5:69]
			key := common.Hex2Bytes(name)
			reader := dpa.Retrieve(key)
			dpaLogger.Debugf("Swarm: Reading %d bytes.", reader.Size())
			mimeType := "application/octet-stream"
			if len(uri) > 70 {
				mimeType = uri[70:]
			}
			w.Header().Set("Content-Type", mimeType)
			http.ServeContent(w, r, name, time.Unix(0, 0), reader)
			dpaLogger.Debugf("Swarm: Object %s returned.", name)
		} else if manifestMatcher.MatchString(uri) {
			dpaLogger.Debugf("Swarm: Structured GET request %s received.", uri)
			name := uri[1:65]
			path := uri[65:] // typically begins with a /
			dpaLogger.Debugf("Swarm: path \"%s\" requested.", path)
			key := common.Hex2Bytes(name)
		MANIFEST_RESOLUTION:
			for {
				manifestReader := dpa.Retrieve(key)
				// TODO check size for oversized manifests
				manifestData := make([]byte, manifestReader.Size())
				size, err := manifestReader.Read(manifestData)
				if int64(size) < manifestReader.Size() {
					dpaLogger.Debugf("Swarm: Manifest %s not found.", name)
					if err == nil {
						http.Error(w, "Manifest retrieval cut short: "+string(size)+"&lt;"+string(manifestReader.Size()),
							http.StatusNotFound)
					} else {
						http.Error(w, err.Error(), http.StatusNotFound)
					}
					return
				}
				dpaLogger.Debugf("Swarm: Manifest %s retrieved.", name)
				man := manifest{}
				err = json.Unmarshal(manifestData, &man)
				if err != nil {
					dpaLogger.Debugf("Swarm: Manifest %s is malformed.", name)
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				} else {
					dpaLogger.Debugf("Swarm: Manifest %s has %d entries.", name, len(man.Entries))
				}
				var mimeType string
				key = nil
				prefix := 0
				status := int16(404)
			MANIFEST_ENTRIES:
				for _, entry := range man.Entries {
					if !hashMatcher.MatchString(entry.Hash) {
						// hash is mandatory
						continue MANIFEST_ENTRIES
					}
					if entry.ContentType == "" {
						// content type defaults to manifest
						entry.ContentType = manifestType
					}
					if entry.Status == 0 {
						// status defaults to 200
						entry.Status = 200
					}
					pathLen := len(entry.Path)
					if len(path) >= pathLen && path[:pathLen] == entry.Path && prefix <= pathLen {
						dpaLogger.Debugf("Swarm: \"%s\" matches \"%s\".", path, entry.Path)
						prefix = pathLen
						key = common.Hex2Bytes(entry.Hash)
						dpaLogger.Debugf("Swarm: Payload hash %064x", key)
						mimeType = entry.ContentType
						status = entry.Status
					}
				}
				if key == nil {
					http.Error(w, "Object "+uri+" not found.", http.StatusNotFound)
					break MANIFEST_RESOLUTION
				} else if mimeType != manifestType {
					w.Header().Set("Content-Type", mimeType)
					dpaLogger.Debugf("Swarm: HTTP Status %d", status)
					w.WriteHeader(int(status))
					reader := dpa.Retrieve(key)
					dpaLogger.Debugf("Swarm: Reading %d bytes.", reader.Size())
					http.ServeContent(w, r, name, time.Unix(0, 0), reader)
					dpaLogger.Debugf("Swarm: Served %s as %s.", mimeType, uri)
					break MANIFEST_RESOLUTION
				} else {
					path = path[prefix:]
					// continue with manifest resolution
				}
			}
		} else {
			http.Error(w, "Object "+uri+" not found.", http.StatusNotFound)
		}
	default:
		http.Error(w, "Method "+r.Method+" is not supported.", http.StatusMethodNotAllowed)
	}
}

func StartHttpServer(dpa *DPA) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, dpa)
	})
	go http.ListenAndServe(port, nil)
	dpaLogger.Infof("Swarm HTTP proxy started.")
}
