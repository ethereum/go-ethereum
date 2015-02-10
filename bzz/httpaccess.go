/*
A simple http server interface to Swarm
*/
package bzz

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethutil"
	"io"
	"net/http"
	"regexp"
	"time"
)

const (
	port = ":8500"
)

var (
	uriMatcher      = regexp.MustCompile("^/raw/[0-9A-Fa-f]{64}$")
	manifestMatcher = regexp.MustCompile("^/[0-9A-Fa-f]{64}")
	hashMatcher     = regexp.MustCompile("^[0-9A-Fa-f]{64}$")
)

type sequentialReader struct {
	reader io.Reader
	pos    int64
	ahead  map[int64](chan bool)
}

type manifestEntry struct {
	Path         string
	Hash         string
	Content_type string
	Status       int16
}

func (self *sequentialReader) ReadAt(target []byte, off int64) (n int, err error) {
	if self.pos != off {
		dpaLogger.Debugf("Swarm: deferred read in POST at position %d, offset %d.",
			self.pos, off)
		wait := make(chan bool)
		self.ahead[off] = wait
		if <-wait {
			// failed read behind
			n = 0
			err = io.ErrUnexpectedEOF
			return
		}
	}
	n, err = self.reader.Read(target)
	dpaLogger.Debugf("Swarm: Read %d bytes into buffer size %d from POST, error %v.",
		n, len(target), err)
	if err != nil {
		for i := range self.ahead {
			self.ahead[i] <- true
			self.ahead[i] = nil
		}
	}
	self.pos += int64(n)
	wait := self.ahead[self.pos]
	if wait != nil {
		dpaLogger.Debugf("Swarm: deferred read in POST at position %d triggered.",
			self.pos)
		self.ahead[self.pos] = nil
		close(wait)
	}
	return
}

func handler(w http.ResponseWriter, r *http.Request, dpa *DPA) {
	uri := r.RequestURI
	switch {
	case r.Method == "POST":
		if uri == "/raw" {
			dpaLogger.Debugf("Swarm: POST request received.")
			key, err := dpa.Store(io.NewSectionReader(&sequentialReader{
				reader: r.Body,
				ahead:  make(map[int64]chan bool),
			}, 0, r.ContentLength))
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
			name := uri[5:]
			key := ethutil.Hex2Bytes(name)
			http.ServeContent(w, r, name+".bin", time.Unix(0, 0), dpa.Retrieve(key))
		} else if manifestMatcher.MatchString(uri) {
			name := uri[1:65]
			path := uri[65:] // typically begins with a /
			key := ethutil.Hex2Bytes(name)
			manifestReader := dpa.Retrieve(key)
			// TODO check size for oversized manifests
			manifest := make([]byte, manifestReader.Size())
			manifestReader.Read(manifest)
			manifestEntries := make([]manifestEntry, 0)
			json.Unmarshal(manifest, &manifestEntries)
			var mimeType string
			key = nil
			prefix := 0
			status := int16(404)
			for i, entry := range manifestEntries {
				if !hashMatcher.MatchString(entry.Hash) {
					// hash is mandatory
					break
				}
				if entry.Content_type == "" {
					// content type defaults to manifest
					entry.Content_type = "application/bzz-manifest+json"
				}
				if entry.Status == 0 {
					// status defaults to 200
					entry.Status = 200
				}
				pathLen := len(entry.Path)
				if len(path) >= pathLen && path[:pathLen] == entry.Path && prefix < pathLen {
					prefix = pathLen
					key = ethutil.Hex2Bytes(entry.Hash)
					mimeType = entry.Content_type
					status = entry.Status
				}
			}
			if key == nil {
				http.Error(w, "Object "+uri+" not found.", http.StatusNotFound)
			} else {
				w.Header().Set("Content-Type", mimeType)
				w.WriteHeader(status)
				http.ServeContent(w, r, "", time.Unix(0, 0), dpa.Retrieve(key))
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
	http.ListenAndServe(port, nil)
}
