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
	"bufio"
	"bytes"
	"encoding/json"
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
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mru"

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
	getListCount    = metrics.NewRegisteredCounter("api.http.get.list.count", nil)
	getListFail     = metrics.NewRegisteredCounter("api.http.get.list.fail", nil)
)

type methodHandler map[string]http.Handler

func (m methodHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	v, ok := m[r.Method]
	if ok {
		v.ServeHTTP(rw, r)
		return
	}
	rw.WriteHeader(http.StatusMethodNotAllowed)
}

func NewServer(api *api.API, corsString string) *Server {
	var allowedOrigins []string
	for _, domain := range strings.Split(corsString, ",") {
		allowedOrigins = append(allowedOrigins, strings.TrimSpace(domain))
	}
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodPost, http.MethodGet, http.MethodDelete, http.MethodPatch, http.MethodPut},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})

	server := &Server{api: api}

	defaultMiddlewares := []Adapter{
		RecoverPanic,
		SetRequestID,
		SetRequestHost,
		InitLoggingResponseWriter,
		ParseURI,
		InstrumentOpenTracing,
	}

	mux := http.NewServeMux()
	mux.Handle("/bzz:/", methodHandler{
		"GET": Adapt(
			http.HandlerFunc(server.HandleBzzGet),
			defaultMiddlewares...,
		),
		"POST": Adapt(
			http.HandlerFunc(server.HandlePostFiles),
			defaultMiddlewares...,
		),
		"DELETE": Adapt(
			http.HandlerFunc(server.HandleDelete),
			defaultMiddlewares...,
		),
	})
	mux.Handle("/bzz-raw:/", methodHandler{
		"GET": Adapt(
			http.HandlerFunc(server.HandleGet),
			defaultMiddlewares...,
		),
		"POST": Adapt(
			http.HandlerFunc(server.HandlePostRaw),
			defaultMiddlewares...,
		),
	})
	mux.Handle("/bzz-immutable:/", methodHandler{
		"GET": Adapt(
			http.HandlerFunc(server.HandleGet),
			defaultMiddlewares...,
		),
	})
	mux.Handle("/bzz-hash:/", methodHandler{
		"GET": Adapt(
			http.HandlerFunc(server.HandleGet),
			defaultMiddlewares...,
		),
	})
	mux.Handle("/bzz-list:/", methodHandler{
		"GET": Adapt(
			http.HandlerFunc(server.HandleGetList),
			defaultMiddlewares...,
		),
	})
	mux.Handle("/bzz-resource:/", methodHandler{
		"GET": Adapt(
			http.HandlerFunc(server.HandleGetResource),
			defaultMiddlewares...,
		),
		"POST": Adapt(
			http.HandlerFunc(server.HandlePostResource),
			defaultMiddlewares...,
		),
	})

	mux.Handle("/", methodHandler{
		"GET": Adapt(
			http.HandlerFunc(server.HandleRootPaths),
			SetRequestID,
			InitLoggingResponseWriter,
		),
	})
	server.Handler = c.Handler(mux)

	return server
}

func (s *Server) ListenAndServe(addr string) error {
	s.listenAddr = addr
	return http.ListenAndServe(addr, s)
}

// browser API for registering bzz url scheme handlers:
// https://developer.mozilla.org/en/docs/Web-based_protocol_handlers
// electron (chromium) api for registering bzz url scheme handlers:
// https://github.com/atom/electron/blob/master/docs/api/protocol.md
type Server struct {
	http.Handler
	api        *api.API
	listenAddr string
}

func (s *Server) HandleBzzGet(w http.ResponseWriter, r *http.Request) {
	log.Debug("handleBzzGet", "ruid", GetRUID(r.Context()), "uri", r.RequestURI)
	if r.Header.Get("Accept") == "application/x-tar" {
		uri := GetURI(r.Context())
		_, credentials, _ := r.BasicAuth()
		reader, err := s.api.GetDirectoryTar(r.Context(), s.api.Decryptor(r.Context(), credentials), uri)
		if err != nil {
			if isDecryptError(err) {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", uri.Address().String()))
				RespondError(w, r, err.Error(), http.StatusUnauthorized)
				return
			}
			RespondError(w, r, fmt.Sprintf("Had an error building the tarball: %v", err), http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		w.Header().Set("Content-Type", "application/x-tar")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, reader)
		return
	}

	s.HandleGetFile(w, r)
}

func (s *Server) HandleRootPaths(w http.ResponseWriter, r *http.Request) {
	switch r.RequestURI {
	case "/":
		RespondTemplate(w, r, "landing-page", "Swarm: Please request a valid ENS or swarm hash with the appropriate bzz scheme", 200)
		return
	case "/robots.txt":
		w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		fmt.Fprintf(w, "User-agent: *\nDisallow: /")
	case "/favicon.ico":
		w.WriteHeader(http.StatusOK)
		w.Write(faviconBytes)
	default:
		RespondError(w, r, "Not Found", http.StatusNotFound)
	}
}

// HandlePostRaw handles a POST request to a raw bzz-raw:/ URI, stores the request
// body in swarm and returns the resulting storage address as a text/plain response
func (s *Server) HandlePostRaw(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	log.Debug("handle.post.raw", "ruid", ruid)

	postRawCount.Inc(1)

	toEncrypt := false
	uri := GetURI(r.Context())
	if uri.Addr == "encrypt" {
		toEncrypt = true
	}

	if uri.Path != "" {
		postRawFail.Inc(1)
		RespondError(w, r, "raw POST request cannot contain a path", http.StatusBadRequest)
		return
	}

	if uri.Addr != "" && uri.Addr != "encrypt" {
		postRawFail.Inc(1)
		RespondError(w, r, "raw POST request addr can only be empty or \"encrypt\"", http.StatusBadRequest)
		return
	}

	if r.Header.Get("Content-Length") == "" {
		postRawFail.Inc(1)
		RespondError(w, r, "missing Content-Length header in request", http.StatusBadRequest)
		return
	}

	addr, _, err := s.api.Store(r.Context(), r.Body, r.ContentLength, toEncrypt)
	if err != nil {
		postRawFail.Inc(1)
		RespondError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug("stored content", "ruid", ruid, "key", addr)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, addr)
}

// HandlePostFiles handles a POST request to
// bzz:/<hash>/<path> which contains either a single file or multiple files
// (either a tar archive or multipart form), adds those files either to an
// existing manifest or to a new manifest under <path> and returns the
// resulting manifest hash as a text/plain response
func (s *Server) HandlePostFiles(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	log.Debug("handle.post.files", "ruid", ruid)
	postFilesCount.Inc(1)

	contentType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		postFilesFail.Inc(1)
		RespondError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	toEncrypt := false
	uri := GetURI(r.Context())
	if uri.Addr == "encrypt" {
		toEncrypt = true
	}

	var addr storage.Address
	if uri.Addr != "" && uri.Addr != "encrypt" {
		addr, err = s.api.Resolve(r.Context(), uri.Addr)
		if err != nil {
			postFilesFail.Inc(1)
			RespondError(w, r, fmt.Sprintf("cannot resolve %s: %s", uri.Addr, err), http.StatusInternalServerError)
			return
		}
		log.Debug("resolved key", "ruid", ruid, "key", addr)
	} else {
		addr, err = s.api.NewManifest(r.Context(), toEncrypt)
		if err != nil {
			postFilesFail.Inc(1)
			RespondError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Debug("new manifest", "ruid", ruid, "key", addr)
	}

	newAddr, err := s.api.UpdateManifest(r.Context(), addr, func(mw *api.ManifestWriter) error {
		switch contentType {
		case "application/x-tar":
			_, err := s.handleTarUpload(r, mw)
			if err != nil {
				RespondError(w, r, fmt.Sprintf("error uploading tarball: %v", err), http.StatusInternalServerError)
				return err
			}
			return nil
		case "multipart/form-data":
			return s.handleMultipartUpload(r, params["boundary"], mw)

		default:
			return s.handleDirectUpload(r, mw)
		}
	})
	if err != nil {
		postFilesFail.Inc(1)
		RespondError(w, r, fmt.Sprintf("cannot create manifest: %s", err), http.StatusInternalServerError)
		return
	}

	log.Debug("stored content", "ruid", ruid, "key", newAddr)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, newAddr)
}

func (s *Server) handleTarUpload(r *http.Request, mw *api.ManifestWriter) (storage.Address, error) {
	log.Debug("handle.tar.upload", "ruid", GetRUID(r.Context()))

	defaultPath := r.URL.Query().Get("defaultpath")

	key, err := s.api.UploadTar(r.Context(), r.Body, GetURI(r.Context()).Path, defaultPath, mw)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (s *Server) handleMultipartUpload(r *http.Request, boundary string, mw *api.ManifestWriter) error {
	ruid := GetRUID(r.Context())
	log.Debug("handle.multipart.upload", "ruid", ruid)
	mr := multipart.NewReader(r.Body, boundary)
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
		uri := GetURI(r.Context())
		path := path.Join(uri.Path, name)
		entry := &api.ManifestEntry{
			Path:        path,
			ContentType: part.Header.Get("Content-Type"),
			Size:        size,
			ModTime:     time.Now(),
		}
		log.Debug("adding path to new manifest", "ruid", ruid, "bytes", entry.Size, "path", entry.Path)
		contentKey, err := mw.AddEntry(r.Context(), reader, entry)
		if err != nil {
			return fmt.Errorf("error adding manifest entry from multipart form: %s", err)
		}
		log.Debug("stored content", "ruid", ruid, "key", contentKey)
	}
}

func (s *Server) handleDirectUpload(r *http.Request, mw *api.ManifestWriter) error {
	ruid := GetRUID(r.Context())
	log.Debug("handle.direct.upload", "ruid", ruid)
	key, err := mw.AddEntry(r.Context(), r.Body, &api.ManifestEntry{
		Path:        GetURI(r.Context()).Path,
		ContentType: r.Header.Get("Content-Type"),
		Mode:        0644,
		Size:        r.ContentLength,
		ModTime:     time.Now(),
	})
	if err != nil {
		return err
	}
	log.Debug("stored content", "ruid", ruid, "key", key)
	return nil
}

// HandleDelete handles a DELETE request to bzz:/<manifest>/<path>, removes
// <path> from <manifest> and returns the resulting manifest hash as a
// text/plain response
func (s *Server) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	uri := GetURI(r.Context())
	log.Debug("handle.delete", "ruid", ruid)
	deleteCount.Inc(1)
	newKey, err := s.api.Delete(r.Context(), uri.Addr, uri.Path)
	if err != nil {
		deleteFail.Inc(1)
		RespondError(w, r, fmt.Sprintf("could not delete from manifest: %v", err), http.StatusInternalServerError)
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
// The POST request admits a JSON structure as defined in the mru package: `mru.updateRequestJSON`
// The requests can be to a) create a resource, b) update a resource or c) both a+b: create a resource and set the initial content
func (s *Server) HandlePostResource(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	log.Debug("handle.post.resource", "ruid", ruid)
	var err error

	// Creation and update must send mru.updateRequestJSON JSON structure
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		RespondError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	var updateRequest mru.Request
	if err := updateRequest.UnmarshalJSON(body); err != nil { // decodes request JSON
		RespondError(w, r, err.Error(), http.StatusBadRequest) //TODO: send different status response depending on error
		return
	}

	if updateRequest.IsUpdate() {
		// Verify that the signature is intact and that the signer is authorized
		// to update this resource
		// Check this early, to avoid creating a resource and then not being able to set its first update.
		if err = updateRequest.Verify(); err != nil {
			RespondError(w, r, err.Error(), http.StatusForbidden)
			return
		}
	}

	if updateRequest.IsNew() {
		err = s.api.ResourceCreate(r.Context(), &updateRequest)
		if err != nil {
			code, err2 := s.translateResourceError(w, r, "resource creation fail", err)
			RespondError(w, r, err2.Error(), code)
			return
		}
	}

	if updateRequest.IsUpdate() {
		_, err = s.api.ResourceUpdate(r.Context(), &updateRequest.SignedResourceUpdate)
		if err != nil {
			RespondError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// at this point both possible operations (create, update or both) were successful
	// so in case it was a new resource, then create a manifest and send it over.

	if updateRequest.IsNew() {
		// we create a manifest so we can retrieve the resource with bzz:// later
		// this manifest has a special "resource type" manifest, and its hash is the key of the mutable resource
		// metadata chunk (rootAddr)
		m, err := s.api.NewResourceManifest(r.Context(), updateRequest.RootAddr().Hex())
		if err != nil {
			RespondError(w, r, fmt.Sprintf("failed to create resource manifest: %v", err), http.StatusInternalServerError)
			return
		}

		// the key to the manifest will be passed back to the client
		// the client can access the root chunk key directly through its Hash member
		// the manifest key should be set as content in the resolver of the ENS name
		// \TODO update manifest key automatically in ENS
		outdata, err := json.Marshal(m)
		if err != nil {
			RespondError(w, r, fmt.Sprintf("failed to create json response: %s", err), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(outdata))
	}
	w.Header().Add("Content-type", "application/json")
}

// Retrieve mutable resource updates:
// bzz-resource://<id> - get latest update
// bzz-resource://<id>/<n> - get latest update on period n
// bzz-resource://<id>/<n>/<m> - get update version m of period n
// bzz-resource://<id>/meta - get metadata and next version information
// <id> = ens name or hash
// TODO: Enable pass maxPeriod parameter
func (s *Server) HandleGetResource(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	uri := GetURI(r.Context())
	log.Debug("handle.get.resource", "ruid", ruid)
	var err error

	// resolve the content key.
	manifestAddr := uri.Address()
	if manifestAddr == nil {
		manifestAddr, err = s.api.Resolve(r.Context(), uri.Addr)
		if err != nil {
			getFail.Inc(1)
			RespondError(w, r, fmt.Sprintf("cannot resolve %s: %s", uri.Addr, err), http.StatusNotFound)
			return
		}
	} else {
		w.Header().Set("Cache-Control", "max-age=2147483648")
	}

	// get the root chunk rootAddr from the manifest
	rootAddr, err := s.api.ResolveResourceManifest(r.Context(), manifestAddr)
	if err != nil {
		getFail.Inc(1)
		RespondError(w, r, fmt.Sprintf("error resolving resource root chunk for %s: %s", uri.Addr, err), http.StatusNotFound)
		return
	}

	log.Debug("handle.get.resource: resolved", "ruid", ruid, "manifestkey", manifestAddr, "rootchunk addr", rootAddr)

	// determine if the query specifies period and version or it is a metadata query
	var params []string
	if len(uri.Path) > 0 {
		if uri.Path == "meta" {
			unsignedUpdateRequest, err := s.api.ResourceNewRequest(r.Context(), rootAddr)
			if err != nil {
				getFail.Inc(1)
				RespondError(w, r, fmt.Sprintf("cannot retrieve resource metadata for rootAddr=%s: %s", rootAddr.Hex(), err), http.StatusNotFound)
				return
			}
			rawResponse, err := unsignedUpdateRequest.MarshalJSON()
			if err != nil {
				RespondError(w, r, fmt.Sprintf("cannot encode unsigned UpdateRequest: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, string(rawResponse))
			return

		}

		params = strings.Split(uri.Path, "/")

	}
	var name string
	var data []byte
	now := time.Now()

	switch len(params) {
	case 0: // latest only
		name, data, err = s.api.ResourceLookup(r.Context(), mru.LookupLatest(rootAddr))
	case 2: // specific period and version
		var version uint64
		var period uint64
		version, err = strconv.ParseUint(params[1], 10, 32)
		if err != nil {
			break
		}
		period, err = strconv.ParseUint(params[0], 10, 32)
		if err != nil {
			break
		}
		name, data, err = s.api.ResourceLookup(r.Context(), mru.LookupVersion(rootAddr, uint32(period), uint32(version)))
	case 1: // last version of specific period
		var period uint64
		period, err = strconv.ParseUint(params[0], 10, 32)
		if err != nil {
			break
		}
		name, data, err = s.api.ResourceLookup(r.Context(), mru.LookupLatestVersionInPeriod(rootAddr, uint32(period)))
	default: // bogus
		err = mru.NewError(storage.ErrInvalidValue, "invalid mutable resource request")
	}

	// any error from the switch statement will end up here
	if err != nil {
		code, err2 := s.translateResourceError(w, r, "mutable resource lookup fail", err)
		RespondError(w, r, err2.Error(), code)
		return
	}

	// All ok, serve the retrieved update
	log.Debug("Found update", "name", name, "ruid", ruid)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, "", now, bytes.NewReader(data))
}

func (s *Server) translateResourceError(w http.ResponseWriter, r *http.Request, supErr string, err error) (int, error) {
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
func (s *Server) HandleGet(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	uri := GetURI(r.Context())
	log.Debug("handle.get", "ruid", ruid, "uri", uri)
	getCount.Inc(1)
	_, pass, _ := r.BasicAuth()

	addr, err := s.api.ResolveURI(r.Context(), uri, pass)
	if err != nil {
		getFail.Inc(1)
		RespondError(w, r, fmt.Sprintf("cannot resolve %s: %s", uri.Addr, err), http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", "max-age=2147483648, immutable") // url was of type bzz://<hex key>/path, so we are sure it is immutable.

	log.Debug("handle.get: resolved", "ruid", ruid, "key", addr)

	// if path is set, interpret <key> as a manifest and return the
	// raw entry at the given path

	etag := common.Bytes2Hex(addr)
	noneMatchEtag := r.Header.Get("If-None-Match")
	w.Header().Set("ETag", fmt.Sprintf("%q", etag)) // set etag to manifest key or raw entry key.
	if noneMatchEtag != "" {
		if bytes.Equal(storage.Address(common.Hex2Bytes(noneMatchEtag)), addr) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// check the root chunk exists by retrieving the file's size
	reader, isEncrypted := s.api.Retrieve(r.Context(), addr)
	if _, err := reader.Size(r.Context(), nil); err != nil {
		getFail.Inc(1)
		RespondError(w, r, fmt.Sprintf("root chunk not found %s: %s", addr, err), http.StatusNotFound)
		return
	}

	w.Header().Set("X-Decrypted", fmt.Sprintf("%v", isEncrypted))

	switch {
	case uri.Raw():
		// allow the request to overwrite the content type using a query
		// parameter
		contentType := "application/octet-stream"
		if typ := r.URL.Query().Get("content_type"); typ != "" {
			contentType = typ
		}
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, r, "", time.Now(), reader)
	case uri.Hash():
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, addr)
	}
}

// HandleGetList handles a GET request to bzz-list:/<manifest>/<path> and returns
// a list of all files contained in <manifest> under <path> grouped into
// common prefixes using "/" as a delimiter
func (s *Server) HandleGetList(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	uri := GetURI(r.Context())
	_, credentials, _ := r.BasicAuth()
	log.Debug("handle.get.list", "ruid", ruid, "uri", uri)
	getListCount.Inc(1)

	// ensure the root path has a trailing slash so that relative URLs work
	if uri.Path == "" && !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	addr, err := s.api.Resolve(r.Context(), uri.Addr)
	if err != nil {
		getListFail.Inc(1)
		RespondError(w, r, fmt.Sprintf("cannot resolve %s: %s", uri.Addr, err), http.StatusNotFound)
		return
	}
	log.Debug("handle.get.list: resolved", "ruid", ruid, "key", addr)

	list, err := s.api.GetManifestList(r.Context(), s.api.Decryptor(r.Context(), credentials), addr, uri.Path)
	if err != nil {
		getListFail.Inc(1)
		if isDecryptError(err) {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", addr.String()))
			RespondError(w, r, err.Error(), http.StatusUnauthorized)
			return
		}
		RespondError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// if the client wants HTML (e.g. a browser) then render the list as a
	// HTML index with relative URLs
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Content-Type", "text/html")
		err := TemplatesMap["bzz-list"].Execute(w, &htmlListData{
			URI: &api.URI{
				Scheme: "bzz",
				Addr:   uri.Addr,
				Path:   uri.Path,
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

// HandleGetFile handles a GET request to bzz://<manifest>/<path> and responds
// with the content of the file at <path> from the given <manifest>
func (s *Server) HandleGetFile(w http.ResponseWriter, r *http.Request) {
	ruid := GetRUID(r.Context())
	uri := GetURI(r.Context())
	_, credentials, _ := r.BasicAuth()
	log.Debug("handle.get.file", "ruid", ruid, "uri", r.RequestURI)
	getFileCount.Inc(1)

	// ensure the root path has a trailing slash so that relative URLs work
	if uri.Path == "" && !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}
	var err error
	manifestAddr := uri.Address()

	if manifestAddr == nil {
		manifestAddr, err = s.api.Resolve(r.Context(), uri.Addr)
		if err != nil {
			getFileFail.Inc(1)
			RespondError(w, r, fmt.Sprintf("cannot resolve %s: %s", uri.Addr, err), http.StatusNotFound)
			return
		}
	} else {
		w.Header().Set("Cache-Control", "max-age=2147483648, immutable") // url was of type bzz://<hex key>/path, so we are sure it is immutable.
	}

	log.Debug("handle.get.file: resolved", "ruid", ruid, "key", manifestAddr)

	reader, contentType, status, contentKey, err := s.api.Get(r.Context(), s.api.Decryptor(r.Context(), credentials), manifestAddr, uri.Path)

	etag := common.Bytes2Hex(contentKey)
	noneMatchEtag := r.Header.Get("If-None-Match")
	w.Header().Set("ETag", fmt.Sprintf("%q", etag)) // set etag to actual content key.
	if noneMatchEtag != "" {
		if bytes.Equal(storage.Address(common.Hex2Bytes(noneMatchEtag)), contentKey) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	if err != nil {
		if isDecryptError(err) {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", manifestAddr))
			RespondError(w, r, err.Error(), http.StatusUnauthorized)
			return
		}

		switch status {
		case http.StatusNotFound:
			getFileNotFound.Inc(1)
			RespondError(w, r, err.Error(), http.StatusNotFound)
		default:
			getFileFail.Inc(1)
			RespondError(w, r, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	//the request results in ambiguous files
	//e.g. /read with readme.md and readinglist.txt available in manifest
	if status == http.StatusMultipleChoices {
		list, err := s.api.GetManifestList(r.Context(), s.api.Decryptor(r.Context(), credentials), manifestAddr, uri.Path)
		if err != nil {
			getFileFail.Inc(1)
			if isDecryptError(err) {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", manifestAddr))
				RespondError(w, r, err.Error(), http.StatusUnauthorized)
				return
			}
			RespondError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debug(fmt.Sprintf("Multiple choices! --> %v", list), "ruid", ruid)
		//show a nice page links to available entries
		ShowMultipleChoices(w, r, list)
		return
	}

	// check the root chunk exists by retrieving the file's size
	if _, err := reader.Size(r.Context(), nil); err != nil {
		getFileNotFound.Inc(1)
		RespondError(w, r, fmt.Sprintf("file not found %s: %s", uri, err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, "", time.Now(), newBufferedReadSeeker(reader, getFileBufferSize))
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

func isDecryptError(err error) bool {
	return strings.Contains(err.Error(), api.ErrDecrypt.Error())
}
