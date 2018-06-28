// Copyright 2018 The go-ethereum Authors
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
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/api/http/messages"
	"github.com/ethereum/go-ethereum/swarm/api/http/views"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type BzzListController struct {
	Api *api.API

	*Controller
}

// Get handles a GET request to bzz-list:/<manifest>/<path> and returns
// a list of all files contained in <manifest> under <path> grouped into
// common prefixes using "/" as a delimiter
func (controller *BzzListController) Get(w http.ResponseWriter, r *messages.Request) {
	log.Debug("handle.get.list", "ruid", r.Ruid, "uri", r.Uri)
	//getListCount.Inc(1)
	// ensure the root path has a trailing slash so that relative URLs work
	if r.Uri.Path == "" && !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, &r.Request, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	addr, err := controller.Api.Resolve(r.Uri)
	if err != nil {
		//getListFail.Inc(1)
		controller.Respond(w, r, fmt.Sprintf("cannot resolve %s: %s", r.Uri.Addr, err), http.StatusNotFound)
		return
	}
	log.Debug("handle.get.list: resolved", "ruid", r.Ruid, "key", addr)

	list, err := controller.GetManifestList(addr, r.Uri.Path)

	if err != nil {
		// getListFail.Inc(1)
		controller.Respond(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// if the client wants HTML (e.g. a browser) then render the list as a
	// HTML index with relative URLs
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Content-Type", "text/html")
		err := views.HtmlListTemplate.Execute(w, &views.HtmlListData{
			URI: &api.URI{
				Scheme: "bzz",
				Addr:   r.Uri.Addr,
				Path:   r.Uri.Path,
			},
			List: &list,
		})
		if err != nil {
			// getListFail.Inc(1)
			log.Error(fmt.Sprintf("error rendering list HTML: %s", err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&list)
}

func (controller *BzzListController) GetManifestList(addr storage.Address, prefix string) (list api.ManifestList, err error) {
	walker, err := controller.Api.NewManifestWalker(addr, nil)
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
