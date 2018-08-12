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

package http

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/api"
)

var (
	htmlCounter      = metrics.NewRegisteredCounter("api.http.errorpage.html.count", nil)
	jsonCounter      = metrics.NewRegisteredCounter("api.http.errorpage.json.count", nil)
	plaintextCounter = metrics.NewRegisteredCounter("api.http.errorpage.plaintext.count", nil)
)

type ResponseParams struct {
	Msg       template.HTML
	Code      int
	Timestamp string
	template  *template.Template
	Details   template.HTML
}

// ShowMultipleChoices is used when a user requests a resource in a manifest which results
// in ambiguous results. It returns a HTML page with clickable links of each of the entry
// in the manifest which fits the request URI ambiguity.
// For example, if the user requests bzz:/<hash>/read and that manifest contains entries
// "readme.md" and "readinglist.txt", a HTML page is returned with this two links.
// This only applies if the manifest has no default entry
func ShowMultipleChoices(w http.ResponseWriter, r *http.Request, list api.ManifestList) {
	log.Debug("ShowMultipleChoices", "ruid", GetRUID(r.Context()), "uri", GetURI(r.Context()))
	msg := ""
	if list.Entries == nil {
		RespondError(w, r, "Could not resolve", http.StatusInternalServerError)
		return
	}
	requestUri := strings.TrimPrefix(r.RequestURI, "/")

	uri, err := api.Parse(requestUri)
	if err != nil {
		RespondError(w, r, "Bad Request", http.StatusBadRequest)
	}

	uri.Scheme = "bzz-list"
	msg += fmt.Sprintf("Disambiguation:<br/>Your request may refer to multiple choices.<br/>Click <a class=\"orange\" href='"+"/"+uri.String()+"'>here</a> if your browser does not redirect you within 5 seconds.<script>setTimeout(\"location.href='%s';\",5000);</script><br/>", "/"+uri.String())
	RespondTemplate(w, r, "error", msg, http.StatusMultipleChoices)
}

func RespondTemplate(w http.ResponseWriter, r *http.Request, templateName, msg string, code int) {
	log.Debug("RespondTemplate", "ruid", GetRUID(r.Context()), "uri", GetURI(r.Context()))
	respond(w, r, &ResponseParams{
		Code:      code,
		Msg:       template.HTML(msg),
		Timestamp: time.Now().Format(time.RFC1123),
		template:  TemplatesMap[templateName],
	})
}

func RespondError(w http.ResponseWriter, r *http.Request, msg string, code int) {
	log.Debug("RespondError", "ruid", GetRUID(r.Context()), "uri", GetURI(r.Context()))
	RespondTemplate(w, r, "error", msg, code)
}

func respond(w http.ResponseWriter, r *http.Request, params *ResponseParams) {

	w.WriteHeader(params.Code)

	if params.Code >= 400 {
		w.Header().Del("Cache-Control")
		w.Header().Del("ETag")
	}

	acceptHeader := r.Header.Get("Accept")
	// this cannot be in a switch since an Accept header can have multiple values: "Accept: */*, text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8"
	if strings.Contains(acceptHeader, "application/json") {
		if err := respondJSON(w, r, params); err != nil {
			RespondError(w, r, "Internal server error", http.StatusInternalServerError)
		}
	} else if strings.Contains(acceptHeader, "text/html") {
		respondHTML(w, r, params)
	} else {
		respondPlaintext(w, r, params) //returns nice errors for curl
	}
}

func respondHTML(w http.ResponseWriter, r *http.Request, params *ResponseParams) {
	htmlCounter.Inc(1)
	log.Debug("respondHTML", "ruid", GetRUID(r.Context()))
	err := params.template.Execute(w, params)
	if err != nil {
		log.Error(err.Error())
	}
}

func respondJSON(w http.ResponseWriter, r *http.Request, params *ResponseParams) error {
	jsonCounter.Inc(1)
	log.Debug("respondJSON", "ruid", GetRUID(r.Context()))
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(params)
}

func respondPlaintext(w http.ResponseWriter, r *http.Request, params *ResponseParams) error {
	plaintextCounter.Inc(1)
	log.Debug("respondPlaintext", "ruid", GetRUID(r.Context()))
	w.Header().Set("Content-Type", "text/plain")
	strToWrite := "Code: " + fmt.Sprintf("%d", params.Code) + "\n"
	strToWrite += "Message: " + string(params.Msg) + "\n"
	strToWrite += "Timestamp: " + params.Timestamp + "\n"
	_, err := w.Write([]byte(strToWrite))
	return err
}
