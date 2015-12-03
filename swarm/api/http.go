/*
A simple http server interface to Swarm
*/
package api

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	rawType = "application/octet-stream"
)

var (
	bzzPrefix       = regexp.MustCompile("^/+bzz:/+")
	rawUrl          = regexp.MustCompile("^/+raw/*")
	trailingSlashes = regexp.MustCompile("/+$")
	// forever         = func() time.Time { return time.Unix(0, 0) }
	forever = time.Now
)

type sequentialReader struct {
	reader io.Reader
	pos    int64
	ahead  map[int64](chan bool)
	lock   sync.Mutex
}

// browser API for registering bzz url scheme handlers:
// https://developer.mozilla.org/en/docs/Web-based_protocol_handlers
// electron (chromium) api for registering bzz url scheme handlers:
// https://github.com/atom/electron/blob/master/docs/api/protocol.md

// starts up http server
func StartHttpServer(api *Api, port string) {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, api)
	})
	go http.ListenAndServe(":"+port, serveMux)
	glog.V(logger.Info).Infof("[BZZ] Swarm HTTP proxy started on localhost:%s", port)
}

func handler(w http.ResponseWriter, r *http.Request, api *Api) {
	requestURL := r.URL
	// This is wrong
	//	if requestURL.Host == "" {
	//		var err error
	//		requestURL, err = url.Parse(r.Referer() + requestURL.String())
	//		if err != nil {
	//			http.Error(w, err.Error(), http.StatusBadRequest)
	//			return
	//		}
	//	}
	glog.V(logger.Debug).Infof("[BZZ] Swarm: HTTP request URL: '%s', Host: '%s', Path: '%s', Referer: '%s', Accept: '%s'", r.RequestURI, requestURL.Host, requestURL.Path, r.Referer(), r.Header.Get("Accept"))
	uri := requestURL.Path
	var raw bool

	// HTTP-based URL protocol handler
	uri = bzzPrefix.ReplaceAllString(uri, "")
	glog.V(logger.Debug).Infof("[BZZ] Swarm: BZZ request URI: '%s'", uri)

	path := rawUrl.ReplaceAllStringFunc(uri, func(string) string {
		raw = true
		return ""
	})

	glog.V(logger.Debug).Infof("[BZZ] Swarm: %s request '%s' received.", r.Method, uri)

	switch {
	case r.Method == "POST" || r.Method == "PUT":
		key, err := api.dpa.Store(io.NewSectionReader(&sequentialReader{
			reader: r.Body,
			ahead:  make(map[int64]chan bool),
		}, 0, r.ContentLength), nil)
		if err == nil {
			glog.V(logger.Debug).Infof("[BZZ] Swarm: Content for %v stored", key.Log())
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Method == "POST" {
			if raw {
				w.Header().Set("Content-Type", "text/plain")
				http.ServeContent(w, r, "", time.Now(), bytes.NewReader([]byte(common.Bytes2Hex(key))))
			} else {
				http.Error(w, "No POST to "+uri+" allowed.", http.StatusBadRequest)
				return
			}
		} else {
			// PUT
			if raw {
				http.Error(w, "No PUT to /raw allowed.", http.StatusBadRequest)
				return
			} else {
				path = regularSlashes(path)
				mime := r.Header.Get("Content-Type")
				// TODO proper root hash separation
				glog.V(logger.Debug).Infof("[BZZ] Modify '%s' to store %v as '%s'.", path, key.Log(), mime)
				newKey, err := api.Modify(path[:64], path[65:], common.Bytes2Hex(key), mime)
				if err == nil {
					glog.V(logger.Debug).Infof("[BZZ] Swarm replaced manifest by '%s'", newKey)
					w.Header().Set("Content-Type", "text/plain")
					http.ServeContent(w, r, "", time.Now(), bytes.NewReader([]byte(newKey)))
				} else {
					http.Error(w, "PUT to "+path+"failed.", http.StatusBadRequest)
					return
				}
			}
		}
	case r.Method == "DELETE":
		if raw {
			http.Error(w, "No DELETE to /raw allowed.", http.StatusBadRequest)
			return
		} else {
			path = regularSlashes(path)
			glog.V(logger.Debug).Infof("[BZZ] Delete '%s'.", path)
			newKey, err := api.Modify(path[:64], path[65:], "", "")
			if err == nil {
				glog.V(logger.Debug).Infof("[BZZ] Swarm replaced manifest by '%s'", newKey)
				w.Header().Set("Content-Type", "text/plain")
				http.ServeContent(w, r, "", time.Now(), bytes.NewReader([]byte(newKey)))
			} else {
				http.Error(w, "DELETE to "+path+"failed.", http.StatusBadRequest)
				return
			}
		}
	case r.Method == "GET" || r.Method == "HEAD":
		path = trailingSlashes.ReplaceAllString(path, "")
		if raw {
			// resolving host
			key, err := api.Resolve(path)
			if err != nil {
				glog.V(logger.Error).Infof("[BZZ] Swarm: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// retrieving content
			reader := api.dpa.Retrieve(key)
			glog.V(logger.Debug).Infof("[BZZ] Swarm: Reading %d bytes.", reader.Size())

			// setting mime type
			qv := requestURL.Query()
			mimeType := qv.Get("content_type")
			if mimeType == "" {
				mimeType = rawType
			}

			w.Header().Set("Content-Type", mimeType)
			http.ServeContent(w, r, uri, forever(), reader)
			glog.V(logger.Debug).Infof("[BZZ] Swarm: Serve raw content '%s' (%d bytes) as '%s'", uri, reader.Size(), mimeType)

			// retrieve path via manifest
		} else {

			glog.V(logger.Debug).Infof("[BZZ] Swarm: Structured GET request '%s' received.", uri)

			// call to api.getPath on uri
			reader, mimeType, status, err := api.getPath(path)
			if err != nil {
				if _, ok := err.(errResolve); ok {
					glog.V(logger.Debug).Infof("[BZZ] Swarm: %v", err)
					status = http.StatusBadRequest
				} else {
					glog.V(logger.Debug).Infof("[BZZ] Swarm: error retrieving '%s': %v", uri, err)
					status = http.StatusNotFound
				}
				http.Error(w, err.Error(), status)
				return
			}

			// set mime type and status headers
			w.Header().Set("Content-Type", mimeType)
			if status > 0 {
				w.WriteHeader(status)
			} else {
				status = 200
			}
			glog.V(logger.Debug).Infof("[BZZ] Swarm: Served '%s' (%d bytes) as '%s' (status code: %v)", uri, reader.Size(), mimeType, status)

			http.ServeContent(w, r, path, forever(), reader)

		}
	default:
		http.Error(w, "Method "+r.Method+" is not supported.", http.StatusMethodNotAllowed)
	}
}

func (self *sequentialReader) ReadAt(target []byte, off int64) (n int, err error) {
	self.lock.Lock()
	// assert self.pos <= off
	if self.pos > off {
		glog.V(logger.Error).Infof("[BZZ] Swarm: non-sequential read attempted from sequentialReader; %d > %d",
			self.pos, off)
		panic("Non-sequential read attempt")
	}
	if self.pos != off {
		glog.V(logger.Debug).Infof("[BZZ] Swarm: deferred read in POST at position %d, offset %d.",
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
		glog.V(logger.Debug).Infof("[BZZ] Swarm: Read %d bytes into buffer size %d from POST, error %v.",
			n, len(target), err)
		if err != nil {
			glog.V(logger.Debug).Infof("[BZZ] Swarm: POST stream's reading terminated with %v.", err)
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
		glog.V(logger.Debug).Infof("[BZZ] Swarm: deferred read in POST at position %d triggered.",
			self.pos)
		delete(self.ahead, self.pos)
		close(wait)
	}
	self.lock.Unlock()
	return localPos, err
}
