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

/*
Show nicely (but simple) formatted HTML error pages (or respond with JSON
if the appropriate `Accept` header is set)) for the http package.
*/
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
	l "github.com/ethereum/go-ethereum/swarm/log"
)

//templateMap holds a mapping of an HTTP error code to a template
var templateMap map[int]*template.Template
var caseErrors []CaseError

//metrics variables
var (
	htmlCounter = metrics.NewRegisteredCounter("api.http.errorpage.html.count", nil)
	jsonCounter = metrics.NewRegisteredCounter("api.http.errorpage.json.count", nil)
)

//parameters needed for formatting the correct HTML page
type ResponseParams struct {
	Msg       string
	Code      int
	Timestamp string
	template  *template.Template
	Details   template.HTML
}

//a custom error case struct that would be used to store validators and
//additional error info to display with client responses.
type CaseError struct {
	Validator func(*Request) bool
	Msg       func(*Request) string
}

//we init the error handling right on boot time, so lookup and http response is fast
func init() {
	initErrHandling()
}

func initErrHandling() {
	//pages are saved as strings - get these strings
	genErrPage := GetGenericErrorPage()
	notFoundPage := GetNotFoundErrorPage()
	multipleChoicesPage := GetMultipleChoicesErrorPage()
	//map the codes to the available pages
	tnames := map[int]string{
		0: genErrPage, //default
		http.StatusBadRequest:          genErrPage,
		http.StatusNotFound:            notFoundPage,
		http.StatusMultipleChoices:     multipleChoicesPage,
		http.StatusInternalServerError: genErrPage,
	}
	templateMap = make(map[int]*template.Template)
	for code, tname := range tnames {
		//assign formatted HTML to the code
		templateMap[code] = template.Must(template.New(fmt.Sprintf("%d", code)).Parse(tname))
	}

	caseErrors = []CaseError{
		{
			Validator: func(r *Request) bool { return r.uri != nil && r.uri.Addr != "" && strings.HasPrefix(r.uri.Addr, "0x") },
			Msg: func(r *Request) string {
				uriCopy := r.uri
				uriCopy.Addr = strings.TrimPrefix(uriCopy.Addr, "0x")
				return fmt.Sprintf(`The requested hash seems to be prefixed with '0x'. You will be redirected to the correct URL within 5 seconds.<br/>
			Please click <a href='%[1]s'>here</a> if your browser does not redirect you.<script>setTimeout("location.href='%[1]s';",5000);</script>`, "/"+uriCopy.String())
			},
		}}
}

//ValidateCaseErrors is a method that process the request object through certain validators
//that assert if certain conditions are met for further information to log as an error
func ValidateCaseErrors(r *Request) string {
	for _, err := range caseErrors {
		if err.Validator(r) {
			return err.Msg(r)
		}
	}

	return ""
}

//ShowMultipeChoices is used when a user requests a resource in a manifest which results
//in ambiguous results. It returns a HTML page with clickable links of each of the entry
//in the manifest which fits the request URI ambiguity.
//For example, if the user requests bzz:/<hash>/read and that manifest contains entries
//"readme.md" and "readinglist.txt", a HTML page is returned with this two links.
//This only applies if the manifest has no default entry
func ShowMultipleChoices(w http.ResponseWriter, req *Request, list api.ManifestList) {
	msg := ""
	if list.Entries == nil {
		Respond(w, req, "Could not resolve", http.StatusInternalServerError)
		return
	}
	//make links relative
	//requestURI comes with the prefix of the ambiguous path, e.g. "read" for "readme.md" and "readinglist.txt"
	//to get clickable links, need to remove the ambiguous path, i.e. "read"
	idx := strings.LastIndex(req.RequestURI, "/")
	if idx == -1 {
		Respond(w, req, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	//remove ambiguous part
	base := req.RequestURI[:idx+1]
	for _, e := range list.Entries {
		//create clickable link for each entry
		msg += "<a href='" + base + e.Path + "'>" + e.Path + "</a><br/>"
	}
	Respond(w, req, msg, http.StatusMultipleChoices)
}

//Respond is used to show an HTML page to a client.
//If there is an `Accept` header of `application/json`, JSON will be returned instead
//The function just takes a string message which will be displayed in the error page.
//The code is used to evaluate which template will be displayed
//(and return the correct HTTP status code)
func Respond(w http.ResponseWriter, req *Request, msg string, code int) {
	additionalMessage := ValidateCaseErrors(req)
	switch code {
	case http.StatusInternalServerError:
		log.Output(msg, log.LvlError, l.CallDepth, "ruid", req.ruid, "code", code)
	case http.StatusMultipleChoices:
		log.Output(msg, log.LvlDebug, l.CallDepth, "ruid", req.ruid, "code", code)
		listURI := api.URI{
			Scheme: "bzz-list",
			Addr:   req.uri.Addr,
			Path:   req.uri.Path,
		}
		additionalMessage = fmt.Sprintf(`<a href="/%s">multiple choices</a>`, listURI.String())
	default:
		log.Output(msg, log.LvlDebug, l.CallDepth, "ruid", req.ruid, "code", code)
	}

	if code >= 400 {
		w.Header().Del("Cache-Control") //avoid sending cache headers for errors!
		w.Header().Del("ETag")
	}

	respond(w, &req.Request, &ResponseParams{
		Code:      code,
		Msg:       msg,
		Details:   template.HTML(additionalMessage),
		Timestamp: time.Now().Format(time.RFC1123),
		template:  getTemplate(code),
	})
}

//evaluate if client accepts html or json response
func respond(w http.ResponseWriter, r *http.Request, params *ResponseParams) {
	w.WriteHeader(params.Code)
	if r.Header.Get("Accept") == "application/json" {
		respondJSON(w, params)
	} else {
		respondHTML(w, params)
	}
}

//return a HTML page
func respondHTML(w http.ResponseWriter, params *ResponseParams) {
	htmlCounter.Inc(1)
	err := params.template.Execute(w, params)
	if err != nil {
		log.Error(err.Error())
	}
}

//return JSON
func respondJSON(w http.ResponseWriter, params *ResponseParams) {
	jsonCounter.Inc(1)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(params)
}

//get the HTML template for a given code
func getTemplate(code int) *template.Template {
	if val, tmpl := templateMap[code]; tmpl {
		return val
	}
	return templateMap[0]
}
