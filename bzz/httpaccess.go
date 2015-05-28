/*
A simple http server interface to Swarm
*/
package bzz

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

const (
	notFoundStatus = 404
	rawType        = "application/octet-stream"
)

var (
	rawUrl          = regexp.MustCompile("^/+raw/*")
	trailingSlashes = regexp.MustCompile("/+$")
)

type sequentialReader struct {
	reader io.Reader
	pos    int64
	ahead  map[int64](chan bool)
	lock   sync.Mutex
}

// starts up http server
// TODO: started by dpa/api rather than backend
func startHttpServer(api *Api, port string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, api)
	})
	go http.ListenAndServe(":"+port, nil)
	dpaLogger.Infof("Swarm HTTP proxy started on localhost:%s", port)
}

func handler(w http.ResponseWriter, r *http.Request, api *Api) {
	dpaLogger.Debugf("request URL: '%s' Host: '%s', Path: '%s'", r.RequestURI, r.URL.Host, r.URL.Path)
	uri := r.URL.Path
	var raw bool
	path := rawUrl.ReplaceAllStringFunc(uri, func(string) string {
		raw = true
		return ""
	})

	switch {
	case r.Method == "POST":
		dpaLogger.Debugf("request URL Host: '%s', Path: '%s'", r.URL.Host, r.URL.Path)
		if raw {
			dpaLogger.Debugf("Swarm: POST request received.")
			key, err := api.dpa.Store(io.NewSectionReader(&sequentialReader{
				reader: r.Body,
				ahead:  make(map[int64]chan bool),
			}, 0, r.ContentLength), nil)
			if err == nil {
				fmt.Fprintf(w, "%064x", key)
				dpaLogger.Debugf("Swarm: Content for '%064x' stored", key)
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "No POST to "+uri+" allowed.", http.StatusBadRequest)
			return
		}

	case r.Method == "GET" || r.Method == "HEAD":
		dpaLogger.Debugf("request URL Host: '%s', Path: '%s'", r.URL.Host, uri)
		path = trailingSlashes.ReplaceAllString(path, "")
		if raw {
			dpaLogger.Debugf("Swarm: Raw GET request '%s' received", uri)

			// resolving host
			key, err := api.Resolve(path)
			if err != nil {
				dpaLogger.Debugf("Swarm: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// retrieving content
			reader := api.dpa.Retrieve(key)
			dpaLogger.Debugf("Swarm: Reading %d bytes.", reader.Size())

			// setting mime type
			qv := r.URL.Query()
			mimeType := qv.Get("content_type")
			if mimeType == "" {
				mimeType = rawType
			}

			w.Header().Set("Content-Type", mimeType)
			http.ServeContent(w, r, uri, time.Unix(0, 0), reader)
			dpaLogger.Debugf("Swarm: Serve raw content '%s' (%d bytes) as '%s'", uri, reader.Size(), mimeType)

			// retrieve path via manifest
		} else {

			dpaLogger.Debugf("Swarm: Structured GET request '%s' received.", uri)

			// call to api.getPath on uri
			reader, mimeType, status, err := api.getPath(path)
			if err != nil {
				if _, ok := err.(errResolve); ok {
					dpaLogger.Debugf("Swarm: %v", err)
					status = http.StatusBadRequest
				} else {
					dpaLogger.Debugf("Swarm: error retrieving '%s': %v", uri, err)
					status = http.StatusNotFound
				}
				http.Error(w, err.Error(), status)
				return
			}

			// set mime type and status headers
			w.Header().Set("Content-Type", mimeType)
			if status > 0 {
				w.WriteHeader(status)
			}
			dpaLogger.Debugf("Swarm: Served '%s' (%d bytes) as '%s' (status code: %v)", uri, reader.Size(), mimeType, w.Header())
			http.ServeContent(w, r, uri, time.Unix(0, 0), reader)

		}
	default:
		http.Error(w, "Method "+r.Method+" is not supported.", http.StatusMethodNotAllowed)
	}
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
