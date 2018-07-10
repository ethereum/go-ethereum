// Copyright 2016 The go-ethereum Authors
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

/*
A simple http server interface to Swarm
*/
package http

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mru"
	"github.com/pborman/uuid"
	"github.com/rs/cors"
)

type resourceResponse struct {
	Manifest storage.Address `json:"manifest"`
	Resource string          `json:"resource"`
	Update   storage.Address `json:"update"`
}

var (
	postRawCount    = metrics.NewRegisteredCounter("api.http.post.raw.count", nil)
	postRawFail     = metrics.NewRegisteredCounter("api.http.post.raw.fail", nil)
	postFilesCount  = metrics.NewRegisteredCounter("api.http.post.files.count", nil)
	postFilesFail   = metrics.NewRegisteredCounter("api.http.post.files.fail", nil)
	deleteCount     = metrics.NewRegisteredCounter("api.http.delete.count", nil)
	deleteFail      = metrics.NewRegisteredCounter("api.http.delete.fail", nil)
	getCount        = metrics.NewRegisteredCounter("api.http.get.count", nil)
	getFail         = metrics.NewRegisteredCounter("api.http.get.fail", nil)
	getFileCount    = metrics.NewRegisteredCounter("api.http.get.file.count", nil)
	getFileNotFound = metrics.NewRegisteredCounter("api.http.get.file.notfound", nil)
	getFileFail     = metrics.NewRegisteredCounter("api.http.get.file.fail", nil)
	getFilesCount   = metrics.NewRegisteredCounter("api.http.get.files.count", nil)
	getFilesFail    = metrics.NewRegisteredCounter("api.http.get.files.fail", nil)
	getListCount    = metrics.NewRegisteredCounter("api.http.get.list.count", nil)
	getListFail     = metrics.NewRegisteredCounter("api.http.get.list.fail", nil)
)

// ServerConfig is the basic configuration needed for the HTTP server and also
// includes CORS settings.
type ServerConfig struct {
	Addr       string
	CorsString string
}

// browser API for registering bzz url scheme handlers:
// https://developer.mozilla.org/en/docs/Web-based_protocol_handlers
// electron (chromium) api for registering bzz url scheme handlers:
// https://github.com/atom/electron/blob/master/docs/api/protocol.md

// starts up http server
func StartHTTPServer(api *api.API, config *ServerConfig) {
	var allowedOrigins []string
	for _, domain := range strings.Split(config.CorsString, ",") {
		allowedOrigins = append(allowedOrigins, strings.TrimSpace(domain))
	}
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"POST", "GET", "DELETE", "PATCH", "PUT"},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	hdlr := c.Handler(NewServer(api))

	go http.ListenAndServe(config.Addr, hdlr)
}

func NewServer(api *api.API) *Server {
	return &Server{api}
}

type Server struct {
	api *api.API
}

// Request wraps http.Request and also includes the parsed bzz URI
type Request struct {
	http.Request

	uri  *api.URI
	ruid string // request unique id
}

// HandlePostRaw handles a POST request to a raw bzz-raw:/ URI, stores the request
// body in swarm and returns the resulting storage address as a text/plain response
func (s *Server) HandlePostRaw(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.post.raw", "ruid", r.ruid)

	postRawCount.Inc(1)

	toEncrypt := false
	if r.uri.Addr == "encrypt" {
		toEncrypt = true
	}

	if r.uri.Path != "" {
		postRawFail.Inc(1)
		Respond(w, r, "raw POST request cannot contain a path", http.StatusBadRequest)
		return
	}

	if r.uri.Addr != "" && r.uri.Addr != "encrypt" {
		postRawFail.Inc(1)
		Respond(w, r, "raw POST request addr can only be empty or \"encrypt\"", http.StatusBadRequest)
		return
	}

	if r.Header.Get("Content-Length") == "" {
		postRawFail.Inc(1)
		Respond(w, r, "missing Content-Length header in request", http.StatusBadRequest)
		return
	}
	addr, _, err := s.api.Store(ctx, r.Body, r.ContentLength, toEncrypt)
	if err != nil {
		postRawFail.Inc(1)
		Respond(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug("stored content", "ruid", r.ruid, "key", addr)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, addr)
}

// HandlePostFiles handles a POST request to
// bzz:/<hash>/<path> which contains either a single file or multiple files
// (either a tar archive or multipart form), adds those files either to an
// existing manifest or to a new manifest under <path> and returns the
// resulting manifest hash as a text/plain response
func (s *Server) HandlePostFiles(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.post.files", "ruid", r.ruid)

	postFilesCount.Inc(1)
	contentType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		postFilesFail.Inc(1)
		Respond(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	toEncrypt := false
	if r.uri.Addr == "encrypt" {
		toEncrypt = true
	}

	var addr storage.Address
	if r.uri.Addr != "" && r.uri.Addr != "encrypt" {
		addr, err = s.api.Resolve(ctx, r.uri)
		if err != nil {
			postFilesFail.Inc(1)
			Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusInternalServerError)
			return
		}
		log.Debug("resolved key", "ruid", r.ruid, "key", addr)
	} else {
		addr, err = s.api.NewManifest(ctx, toEncrypt)
		if err != nil {
			postFilesFail.Inc(1)
			Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Debug("new manifest", "ruid", r.ruid, "key", addr)
	}

	newAddr, err := s.updateManifest(ctx, addr, func(mw *api.ManifestWriter) error {
		switch contentType {

		case "application/x-tar":
			return s.handleTarUpload(ctx, r, mw)

		case "multipart/form-data":
			return s.handleMultipartUpload(ctx, r, params["boundary"], mw)

		default:
			return s.handleDirectUpload(ctx, r, mw)
		}
	})
	if err != nil {
		postFilesFail.Inc(1)
		Respond(w, r, fmt.Sprintf("cannot create manifest: %s", err), http.StatusInternalServerError)
		return
	}

	log.Debug("stored content", "ruid", r.ruid, "key", newAddr)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, newAddr)
}

func (s *Server) handleTarUpload(ctx context.Context, req *Request, mw *api.ManifestWriter) error {
	log.Debug("handle.tar.upload", "ruid", req.ruid)
	tr := tar.NewReader(req.Body)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("error reading tar stream: %s", err)
		}

		// only store regular files
		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}

		// add the entry under the path from the request
		path := path.Join(req.uri.Path, hdr.Name)
		entry := &api.ManifestEntry{
			Path:        path,
			ContentType: hdr.Xattrs["user.swarm.content-type"],
			Mode:        hdr.Mode,
			Size:        hdr.Size,
			ModTime:     hdr.ModTime,
		}
		log.Debug("adding path to new manifest", "ruid", req.ruid, "bytes", entry.Size, "path", entry.Path)
		contentKey, err := mw.AddEntry(ctx, tr, entry)
		if err != nil {
			return fmt.Errorf("error adding manifest entry from tar stream: %s", err)
		}
		log.Debug("stored content", "ruid", req.ruid, "key", contentKey)
	}
}

func (s *Server) handleMultipartUpload(ctx context.Context, req *Request, boundary string, mw *api.ManifestWriter) error {
	log.Debug("handle.multipart.upload", "ruid", req.ruid)
	mr := multipart.NewReader(req.Body, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("error reading multipart form: %s", err)
		}

		var size int64
		var reader io.Reader = part
		if contentLength := part.Header.Get("Content-Length"); contentLength != "" {
			size, err = strconv.ParseInt(contentLength, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing multipart content length: %s", err)
			}
			reader = part
		} else {
			// copy the part to a tmp file to get its size
			tmp, err := ioutil.TempFile("", "swarm-multipart")
			if err != nil {
				return err
			}
			defer os.Remove(tmp.Name())
			defer tmp.Close()
			size, err = io.Copy(tmp, part)
			if err != nil {
				return fmt.Errorf("error copying multipart content: %s", err)
			}
			if _, err := tmp.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("error copying multipart content: %s", err)
			}
			reader = tmp
		}

		// add the entry under the path from the request
		name := part.FileName()
		if name == "" {
			name = part.FormName()
		}
		path := path.Join(req.uri.Path, name)
		entry := &api.ManifestEntry{
			Path:        path,
			ContentType: part.Header.Get("Content-Type"),
			Size:        size,
			ModTime:     time.Now(),
		}
		log.Debug("adding path to new manifest", "ruid", req.ruid, "bytes", entry.Size, "path", entry.Path)
		contentKey, err := mw.AddEntry(ctx, reader, entry)
		if err != nil {
			return fmt.Errorf("error adding manifest entry from multipart form: %s", err)
		}
		log.Debug("stored content", "ruid", req.ruid, "key", contentKey)
	}
}

func (s *Server) handleDirectUpload(ctx context.Context, req *Request, mw *api.ManifestWriter) error {
	log.Debug("handle.direct.upload", "ruid", req.ruid)
	key, err := mw.AddEntry(ctx, req.Body, &api.ManifestEntry{
		Path:        req.uri.Path,
		ContentType: req.Header.Get("Content-Type"),
		Mode:        0644,
		Size:        req.ContentLength,
		ModTime:     time.Now(),
	})
	if err != nil {
		return err
	}
	log.Debug("stored content", "ruid", req.ruid, "key", key)
	return nil
}

// HandleDelete handles a DELETE request to bzz:/<manifest>/<path>, removes
// <path> from <manifest> and returns the resulting manifest hash as a
// text/plain response
func (s *Server) HandleDelete(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.delete", "ruid", r.ruid)

	deleteCount.Inc(1)
	key, err := s.api.Resolve(ctx, r.uri)
	if err != nil {
		deleteFail.Inc(1)
		Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusInternalServerError)
		return
	}

	newKey, err := s.updateManifest(ctx, key, func(mw *api.ManifestWriter) error {
		log.Debug(fmt.Sprintf("removing %s from manifest %s", r.uri.Path, key.Log()), "ruid", r.ruid)
		return mw.RemoveEntry(r.uri.Path)
	})
	if err != nil {
		deleteFail.Inc(1)
		Respond(w, r, fmt.Sprintf("cannot update manifest: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, newKey)
}

// Parses a resource update post url to corresponding action
// possible combinations:
// /			add multihash update to existing hash
// /raw 		add raw update to existing hash
// /#			create new resource with first update as mulitihash
// /raw/#		create new resource with first update raw
func resourcePostMode(path string) (isRaw bool, frequency uint64, err error) {
	re, err := regexp.Compile("^(raw)?/?([0-9]+)?$")
	if err != nil {
		return isRaw, frequency, err
	}
	m := re.FindAllStringSubmatch(path, 2)
	var freqstr = "0"
	if len(m) > 0 {
		if m[0][1] != "" {
			isRaw = true
		}
		if m[0][2] != "" {
			freqstr = m[0][2]
		}
	} else if len(path) > 0 {
		return isRaw, frequency, fmt.Errorf("invalid path")
	}
	frequency, err = strconv.ParseUint(freqstr, 10, 64)
	return isRaw, frequency, err
}

// Handles creation of new mutable resources and adding updates to existing mutable resources
// There are two types of updates available, "raw" and "multihash."
// If the latter is used, a subsequent bzz:// GET call to the manifest of the resource will return
// the page that the multihash is pointing to, as if it held a normal swarm content manifest
//
// The resource name will be verbatim what is passed as the address part of the url.
// For example, if a POST is made to /bzz-resource:/foo.eth/raw/13 a new resource with frequency 13
// and name "foo.eth" will be created
func (s *Server) HandlePostResource(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.post.resource", "ruid", r.ruid)
	var err error
	var addr storage.Address
	var name string
	var outdata []byte
	isRaw, frequency, err := resourcePostMode(r.uri.Path)
	if err != nil {
		Respond(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// new mutable resource creation will always have a frequency field larger than 0
	if frequency > 0 {

		name = r.uri.Addr

		// the key is the content addressed root chunk holding mutable resource metadata information
		addr, err = s.api.ResourceCreate(r.Context(), name, frequency)
		if err != nil {
			code, err2 := s.translateResourceError(w, r, "resource creation fail", err)

			Respond(w, r, err2.Error(), code)
			return
		}

		// we create a manifest so we can retrieve the resource with bzz:// later
		// this manifest has a special "resource type" manifest, and its hash is the key of the mutable resource
		// root chunk
		m, err := s.api.NewResourceManifest(ctx, addr.Hex())
		if err != nil {
			Respond(w, r, fmt.Sprintf("failed to create resource manifest: %v", err), http.StatusInternalServerError)
			return
		}

		// the key to the manifest will be passed back to the client
		// the client can access the root chunk key directly through its Hash member
		// the manifest key should be set as content in the resolver of the ENS name
		// \TODO update manifest key automatically in ENS
		outdata, err = json.Marshal(m)
		if err != nil {
			Respond(w, r, fmt.Sprintf("failed to create json response: %s", err), http.StatusInternalServerError)
			return
		}
	} else {
		// to update the resource through http we need to retrieve the key for the mutable resource root chunk
		// that means that we retrieve the manifest and inspect its Hash member.
		manifestAddr := r.uri.Address()
		if manifestAddr == nil {
			manifestAddr, err = s.api.Resolve(ctx, r.uri)
			if err != nil {
				getFail.Inc(1)
				Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusNotFound)
				return
			}
		} else {
			w.Header().Set("Cache-Control", "max-age=2147483648")
		}

		// get the root chunk key from the manifest
		addr, err = s.api.ResolveResourceManifest(ctx, manifestAddr)
		if err != nil {
			getFail.Inc(1)
			Respond(w, r, fmt.Sprintf("error resolving resource root chunk for %s: %s", r.uri.Addr, err), http.StatusNotFound)
			return
		}

		log.Debug("handle.post.resource: resolved", "ruid", r.ruid, "manifestkey", manifestAddr, "rootchunkkey", addr)

		name, _, err = s.api.ResourceLookup(r.Context(), addr, 0, 0, &mru.LookupParams{})
		if err != nil {
			Respond(w, r, err.Error(), http.StatusNotFound)
			return
		}
	}

	// Creation and update must send data aswell. This data constitutes the update data itself.
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Respond(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Multihash will be passed as hex-encoded data, so we need to parse this to bytes
	if isRaw {
		_, _, _, err = s.api.ResourceUpdate(r.Context(), name, data)
		if err != nil {
			Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		bytesdata, err := hexutil.Decode(string(data))
		if err != nil {
			Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		_, _, _, err = s.api.ResourceUpdateMultihash(r.Context(), name, bytesdata)
		if err != nil {
			Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// If we have data to return, write this now
	// \TODO there should always be data to return here
	if len(outdata) > 0 {
		w.Header().Add("Content-type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(outdata))
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Retrieve mutable resource updates:
// bzz-resource://<id> - get latest update
// bzz-resource://<id>/<n> - get latest update on period n
// bzz-resource://<id>/<n>/<m> - get update version m of period n
// <id> = ens name or hash
func (s *Server) HandleGetResource(ctx context.Context, w http.ResponseWriter, r *Request) {
	s.handleGetResource(ctx, w, r)
}

// TODO: Enable pass maxPeriod parameter
func (s *Server) handleGetResource(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.get.resource", "ruid", r.ruid)
	var err error

	// resolve the content key.
	manifestAddr := r.uri.Address()
	if manifestAddr == nil {
		manifestAddr, err = s.api.Resolve(ctx, r.uri)
		if err != nil {
			getFail.Inc(1)
			Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusNotFound)
			return
		}
	} else {
		w.Header().Set("Cache-Control", "max-age=2147483648")
	}

	// get the root chunk key from the manifest
	key, err := s.api.ResolveResourceManifest(ctx, manifestAddr)
	if err != nil {
		getFail.Inc(1)
		Respond(w, r, fmt.Sprintf("error resolving resource root chunk for %s: %s", r.uri.Addr, err), http.StatusNotFound)
		return
	}

	log.Debug("handle.get.resource: resolved", "ruid", r.ruid, "manifestkey", manifestAddr, "rootchunk key", key)

	// determine if the query specifies period and version
	var params []string
	if len(r.uri.Path) > 0 {
		params = strings.Split(r.uri.Path, "/")
	}
	var name string
	var period uint64
	var version uint64
	var data []byte
	now := time.Now()

	switch len(params) {
	case 0: // latest only
		name, data, err = s.api.ResourceLookup(r.Context(), key, 0, 0, nil)
	case 2: // specific period and version
		version, err = strconv.ParseUint(params[1], 10, 32)
		if err != nil {
			break
		}
		period, err = strconv.ParseUint(params[0], 10, 32)
		if err != nil {
			break
		}
		name, data, err = s.api.ResourceLookup(r.Context(), key, uint32(period), uint32(version), nil)
	case 1: // last version of specific period
		period, err = strconv.ParseUint(params[0], 10, 32)
		if err != nil {
			break
		}
		name, data, err = s.api.ResourceLookup(r.Context(), key, uint32(period), uint32(version), nil)
	default: // bogus
		err = mru.NewError(storage.ErrInvalidValue, "invalid mutable resource request")
	}

	// any error from the switch statement will end up here
	if err != nil {
		code, err2 := s.translateResourceError(w, r, "mutable resource lookup fail", err)
		Respond(w, r, err2.Error(), code)
		return
	}

	// All ok, serve the retrieved update
	log.Debug("Found update", "name", name, "ruid", r.ruid)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, &r.Request, "", now, bytes.NewReader(data))
}

func (s *Server) translateResourceError(w http.ResponseWriter, r *Request, supErr string, err error) (int, error) {
	code := 0
	defaultErr := fmt.Errorf("%s: %v", supErr, err)
	rsrcErr, ok := err.(*mru.Error)
	if !ok && rsrcErr != nil {
		code = rsrcErr.Code()
	}
	switch code {
	case storage.ErrInvalidValue:
		return http.StatusBadRequest, defaultErr
	case storage.ErrNotFound, storage.ErrNotSynced, storage.ErrNothingToReturn, storage.ErrInit:
		return http.StatusNotFound, defaultErr
	case storage.ErrUnauthorized, storage.ErrInvalidSignature:
		return http.StatusUnauthorized, defaultErr
	case storage.ErrDataOverflow:
		return http.StatusRequestEntityTooLarge, defaultErr
	}

	return http.StatusInternalServerError, defaultErr
}

// HandleGet handles a GET request to
// - bzz-raw://<key> and responds with the raw content stored at the
//   given storage key
// - bzz-hash://<key> and responds with the hash of the content stored
//   at the given storage key as a text/plain response
func (s *Server) HandleGet(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.get", "ruid", r.ruid, "uri", r.uri)
	getCount.Inc(1)
	var err error
	addr := r.uri.Address()
	if addr == nil {
		addr, err = s.api.Resolve(ctx, r.uri)
		if err != nil {
			getFail.Inc(1)
			Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusNotFound)
			return
		}
	} else {
		w.Header().Set("Cache-Control", "max-age=2147483648, immutable") // url was of type bzz://<hex key>/path, so we are sure it is immutable.
	}

	log.Debug("handle.get: resolved", "ruid", r.ruid, "key", addr)

	// if path is set, interpret <key> as a manifest and return the
	// raw entry at the given path
	if r.uri.Path != "" {
		walker, err := s.api.NewManifestWalker(ctx, addr, nil)
		if err != nil {
			getFail.Inc(1)
			Respond(w, r, fmt.Sprintf("%s is not a manifest", addr), http.StatusBadRequest)
			return
		}
		var entry *api.ManifestEntry
		walker.Walk(func(e *api.ManifestEntry) error {
			// if the entry matches the path, set entry and stop
			// the walk
			if e.Path == r.uri.Path {
				entry = e
				// return an error to cancel the walk
				return errors.New("found")
			}

			// ignore non-manifest files
			if e.ContentType != api.ManifestType {
				return nil
			}

			// if the manifest's path is a prefix of the
			// requested path, recurse into it by returning
			// nil and continuing the walk
			if strings.HasPrefix(r.uri.Path, e.Path) {
				return nil
			}

			return api.ErrSkipManifest
		})
		if entry == nil {
			getFail.Inc(1)
			Respond(w, r, fmt.Sprintf("manifest entry could not be loaded"), http.StatusNotFound)
			return
		}
		addr = storage.Address(common.Hex2Bytes(entry.Hash))
	}
	etag := common.Bytes2Hex(addr)
	noneMatchEtag := r.Header.Get("If-None-Match")
	w.Header().Set("ETag", fmt.Sprintf("%q", etag)) // set etag to manifest key or raw entry key.
	if noneMatchEtag != "" {
		if bytes.Equal(storage.Address(common.Hex2Bytes(noneMatchEtag)), addr) {
			Respond(w, r, "Not Modified", http.StatusNotModified)
			return
		}
	}

	// check the root chunk exists by retrieving the file's size
	reader, isEncrypted := s.api.Retrieve(ctx, addr)
	if _, err := reader.Size(nil); err != nil {
		getFail.Inc(1)
		Respond(w, r, fmt.Sprintf("root chunk not found %s: %s", addr, err), http.StatusNotFound)
		return
	}

	w.Header().Set("X-Decrypted", fmt.Sprintf("%v", isEncrypted))

	switch {
	case r.uri.Raw():
		// allow the request to overwrite the content type using a query
		// parameter
		contentType := "application/octet-stream"
		if typ := r.URL.Query().Get("content_type"); typ != "" {
			contentType = typ
		}
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, &r.Request, "", time.Now(), reader)
	case r.uri.Hash():
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, addr)
	}
}

// HandleGetFiles handles a GET request to bzz:/<manifest> with an Accept
// header of "application/x-tar" and returns a tar stream of all files
// contained in the manifest
func (s *Server) HandleGetFiles(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.get.files", "ruid", r.ruid, "uri", r.uri)
	getFilesCount.Inc(1)
	if r.uri.Path != "" {
		getFilesFail.Inc(1)
		Respond(w, r, "files request cannot contain a path", http.StatusBadRequest)
		return
	}

	addr, err := s.api.Resolve(ctx, r.uri)
	if err != nil {
		getFilesFail.Inc(1)
		Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusNotFound)
		return
	}
	log.Debug("handle.get.files: resolved", "ruid", r.ruid, "key", addr)

	walker, err := s.api.NewManifestWalker(ctx, addr, nil)
	if err != nil {
		getFilesFail.Inc(1)
		Respond(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	tw := tar.NewWriter(w)
	defer tw.Close()
	w.Header().Set("Content-Type", "application/x-tar")
	w.WriteHeader(http.StatusOK)

	err = walker.Walk(func(entry *api.ManifestEntry) error {
		// ignore manifests (walk will recurse into them)
		if entry.ContentType == api.ManifestType {
			return nil
		}

		// retrieve the entry's key and size
		reader, isEncrypted := s.api.Retrieve(ctx, storage.Address(common.Hex2Bytes(entry.Hash)))
		size, err := reader.Size(nil)
		if err != nil {
			return err
		}
		w.Header().Set("X-Decrypted", fmt.Sprintf("%v", isEncrypted))

		// write a tar header for the entry
		hdr := &tar.Header{
			Name:    entry.Path,
			Mode:    entry.Mode,
			Size:    size,
			ModTime: entry.ModTime,
			Xattrs: map[string]string{
				"user.swarm.content-type": entry.ContentType,
			},
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		// copy the file into the tar stream
		n, err := io.Copy(tw, io.LimitReader(reader, hdr.Size))
		if err != nil {
			return err
		} else if n != size {
			return fmt.Errorf("error writing %s: expected %d bytes but sent %d", entry.Path, size, n)
		}

		return nil
	})
	if err != nil {
		getFilesFail.Inc(1)
		log.Error(fmt.Sprintf("error generating tar stream: %s", err))
	}
}

// HandleGetList handles a GET request to bzz-list:/<manifest>/<path> and returns
// a list of all files contained in <manifest> under <path> grouped into
// common prefixes using "/" as a delimiter
func (s *Server) HandleGetList(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.get.list", "ruid", r.ruid, "uri", r.uri)
	getListCount.Inc(1)
	// ensure the root path has a trailing slash so that relative URLs work
	if r.uri.Path == "" && !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, &r.Request, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	addr, err := s.api.Resolve(ctx, r.uri)
	if err != nil {
		getListFail.Inc(1)
		Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusNotFound)
		return
	}
	log.Debug("handle.get.list: resolved", "ruid", r.ruid, "key", addr)

	list, err := s.getManifestList(ctx, addr, r.uri.Path)

	if err != nil {
		getListFail.Inc(1)
		Respond(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// if the client wants HTML (e.g. a browser) then render the list as a
	// HTML index with relative URLs
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Content-Type", "text/html")
		err := htmlListTemplate.Execute(w, &htmlListData{
			URI: &api.URI{
				Scheme: "bzz",
				Addr:   r.uri.Addr,
				Path:   r.uri.Path,
			},
			List: &list,
		})
		if err != nil {
			getListFail.Inc(1)
			log.Error(fmt.Sprintf("error rendering list HTML: %s", err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&list)
}

func (s *Server) getManifestList(ctx context.Context, addr storage.Address, prefix string) (list api.ManifestList, err error) {
	walker, err := s.api.NewManifestWalker(ctx, addr, nil)
	if err != nil {
		return
	}

	err = walker.Walk(func(entry *api.ManifestEntry) error {
		// handle non-manifest files
		if entry.ContentType != api.ManifestType {
			// ignore the file if it doesn't have the specified prefix
			if !strings.HasPrefix(entry.Path, prefix) {
				return nil
			}

			// if the path after the prefix contains a slash, add a
			// common prefix to the list, otherwise add the entry
			suffix := strings.TrimPrefix(entry.Path, prefix)
			if index := strings.Index(suffix, "/"); index > -1 {
				list.CommonPrefixes = append(list.CommonPrefixes, prefix+suffix[:index+1])
				return nil
			}
			if entry.Path == "" {
				entry.Path = "/"
			}
			list.Entries = append(list.Entries, entry)
			return nil
		}

		// if the manifest's path is a prefix of the specified prefix
		// then just recurse into the manifest by returning nil and
		// continuing the walk
		if strings.HasPrefix(prefix, entry.Path) {
			return nil
		}

		// if the manifest's path has the specified prefix, then if the
		// path after the prefix contains a slash, add a common prefix
		// to the list and skip the manifest, otherwise recurse into
		// the manifest by returning nil and continuing the walk
		if strings.HasPrefix(entry.Path, prefix) {
			suffix := strings.TrimPrefix(entry.Path, prefix)
			if index := strings.Index(suffix, "/"); index > -1 {
				list.CommonPrefixes = append(list.CommonPrefixes, prefix+suffix[:index+1])
				return api.ErrSkipManifest
			}
			return nil
		}

		// the manifest neither has the prefix or needs recursing in to
		// so just skip it
		return api.ErrSkipManifest
	})

	return list, nil
}

// HandleGetFile handles a GET request to bzz://<manifest>/<path> and responds
// with the content of the file at <path> from the given <manifest>
func (s *Server) HandleGetFile(ctx context.Context, w http.ResponseWriter, r *Request) {
	log.Debug("handle.get.file", "ruid", r.ruid)
	getFileCount.Inc(1)
	// ensure the root path has a trailing slash so that relative URLs work
	if r.uri.Path == "" && !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, &r.Request, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}
	var err error
	manifestAddr := r.uri.Address()

	if manifestAddr == nil {
		manifestAddr, err = s.api.Resolve(ctx, r.uri)
		if err != nil {
			getFileFail.Inc(1)
			Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.uri.Addr, err), http.StatusNotFound)
			return
		}
	} else {
		w.Header().Set("Cache-Control", "max-age=2147483648, immutable") // url was of type bzz://<hex key>/path, so we are sure it is immutable.
	}

	log.Debug("handle.get.file: resolved", "ruid", r.ruid, "key", manifestAddr)

	reader, contentType, status, contentKey, err := s.api.Get(ctx, manifestAddr, r.uri.Path)

	etag := common.Bytes2Hex(contentKey)
	noneMatchEtag := r.Header.Get("If-None-Match")
	w.Header().Set("ETag", fmt.Sprintf("%q", etag)) // set etag to actual content key.
	if noneMatchEtag != "" {
		if bytes.Equal(storage.Address(common.Hex2Bytes(noneMatchEtag)), contentKey) {
			Respond(w, r, "Not Modified", http.StatusNotModified)
			return
		}
	}

	if err != nil {
		switch status {
		case http.StatusNotFound:
			getFileNotFound.Inc(1)
			Respond(w, r, err.Error(), http.StatusNotFound)
		default:
			getFileFail.Inc(1)
			Respond(w, r, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	//the request results in ambiguous files
	//e.g. /read with readme.md and readinglist.txt available in manifest
	if status == http.StatusMultipleChoices {
		list, err := s.getManifestList(ctx, manifestAddr, r.uri.Path)

		if err != nil {
			getFileFail.Inc(1)
			Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debug(fmt.Sprintf("Multiple choices! --> %v", list), "ruid", r.ruid)
		//show a nice page links to available entries
		ShowMultipleChoices(w, r, list)
		return
	}

	// check the root chunk exists by retrieving the file's size
	if _, err := reader.Size(nil); err != nil {
		getFileNotFound.Inc(1)
		Respond(w, r, fmt.Sprintf("file not found %s: %s", r.uri, err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, &r.Request, "", time.Now(), newBufferedReadSeeker(reader, getFileBufferSize))
}

// The size of buffer used for bufio.Reader on LazyChunkReader passed to
// http.ServeContent in HandleGetFile.
// Warning: This value influences the number of chunk requests and chunker join goroutines
// per file request.
// Recommended value is 4 times the io.Copy default buffer value which is 32kB.
const getFileBufferSize = 4 * 32 * 1024

// bufferedReadSeeker wraps bufio.Reader to expose Seek method
// from the provied io.ReadSeeker in newBufferedReadSeeker.
type bufferedReadSeeker struct {
	r io.Reader
	s io.Seeker
}

// newBufferedReadSeeker creates a new instance of bufferedReadSeeker,
// out of io.ReadSeeker. Argument `size` is the size of the read buffer.
func newBufferedReadSeeker(readSeeker io.ReadSeeker, size int) bufferedReadSeeker {
	return bufferedReadSeeker{
		r: bufio.NewReaderSize(readSeeker, size),
		s: readSeeker,
	}
}

func (b bufferedReadSeeker) Read(p []byte) (n int, err error) {
	return b.r.Read(p)
}

func (b bufferedReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return b.s.Seek(offset, whence)
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	defer metrics.GetOrRegisterResettingTimer(fmt.Sprintf("http.request.%s.time", r.Method), nil).UpdateSince(time.Now())
	req := &Request{Request: *r, ruid: uuid.New()[:8]}
	metrics.GetOrRegisterCounter(fmt.Sprintf("http.request.%s", r.Method), nil).Inc(1)
	log.Info("serving request", "ruid", req.ruid, "method", r.Method, "url", r.RequestURI)

	// wrapping the ResponseWriter, so that we get the response code set by http.ServeContent
	w := newLoggingResponseWriter(rw)

	if r.RequestURI == "/" && strings.Contains(r.Header.Get("Accept"), "text/html") {

		err := landingPageTemplate.Execute(w, nil)
		if err != nil {
			log.Error(fmt.Sprintf("error rendering landing page: %s", err))
		}
		return
	}

	if r.URL.Path == "/robots.txt" {
		w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		fmt.Fprintf(w, "User-agent: *\nDisallow: /")
		return
	}

	if r.RequestURI == "/" && strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("Welcome to Swarm!")
		return
	}

	uri, err := api.Parse(strings.TrimLeft(r.URL.Path, "/"))
	if err != nil {
		Respond(w, req, fmt.Sprintf("invalid URI %q", r.URL.Path), http.StatusBadRequest)
		return
	}

	req.uri = uri

	log.Debug("parsed request path", "ruid", req.ruid, "method", req.Method, "uri.Addr", req.uri.Addr, "uri.Path", req.uri.Path, "uri.Scheme", req.uri.Scheme)

	switch r.Method {
	case "POST":
		if uri.Raw() {
			log.Debug("handlePostRaw")
			s.HandlePostRaw(ctx, w, req)
		} else if uri.Resource() {
			log.Debug("handlePostResource")
			s.HandlePostResource(ctx, w, req)
		} else if uri.Immutable() || uri.List() || uri.Hash() {
			log.Debug("POST not allowed on immutable, list or hash")
			Respond(w, req, fmt.Sprintf("POST method on scheme %s not allowed", uri.Scheme), http.StatusMethodNotAllowed)
		} else {
			log.Debug("handlePostFiles")
			s.HandlePostFiles(ctx, w, req)
		}

	case "PUT":
		Respond(w, req, fmt.Sprintf("PUT method to %s not allowed", uri), http.StatusBadRequest)
		return

	case "DELETE":
		if uri.Raw() {
			Respond(w, req, fmt.Sprintf("DELETE method to %s not allowed", uri), http.StatusBadRequest)
			return
		}
		s.HandleDelete(ctx, w, req)

	case "GET":

		if uri.Resource() {
			s.HandleGetResource(ctx, w, req)
			return
		}

		if uri.Raw() || uri.Hash() {
			s.HandleGet(ctx, w, req)
			return
		}

		if uri.List() {
			s.HandleGetList(ctx, w, req)
			return
		}

		if r.Header.Get("Accept") == "application/x-tar" {
			s.HandleGetFiles(ctx, w, req)
			return
		}

		s.HandleGetFile(ctx, w, req)

	default:
		Respond(w, req, fmt.Sprintf("%s method is not supported", r.Method), http.StatusMethodNotAllowed)
	}

	log.Info("served response", "ruid", req.ruid, "code", w.statusCode)
}

func (s *Server) updateManifest(ctx context.Context, addr storage.Address, update func(mw *api.ManifestWriter) error) (storage.Address, error) {
	mw, err := s.api.NewManifestWriter(ctx, addr, nil)
	if err != nil {
		return nil, err
	}

	if err := update(mw); err != nil {
		return nil, err
	}

	addr, err = mw.Store()
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("generated manifest %s", addr))
	return addr, nil
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
