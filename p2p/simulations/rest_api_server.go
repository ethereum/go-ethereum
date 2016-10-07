package simulations

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type Controller interface {
	Resource(id string) (Controller, error)
	Handle(method string) (returnHandler, error)
	SetResource(id string, c Controller)
}

// starts up http server
func StartRestApiServer(port string, c Controller) {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, c)
	})
	go http.ListenAndServe(":"+port, serveMux)
	glog.V(logger.Info).Infof("Swarm Network Controller HTTP server started on localhost:%s", port)
}

func handle(w http.ResponseWriter, r *http.Request, c Controller) {
	requestURL := r.URL
	glog.V(logger.Debug).Infof("HTTP %s request URL: '%s', Host: '%s', Path: '%s', Referer: '%s', Accept: '%s'", r.Method, r.RequestURI, requestURL.Host, requestURL.Path, r.Referer(), r.Header.Get("Accept"))
	uri := requestURL.Path
	w.Header().Set("Content-Type", "text/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	defer r.Body.Close()
	parts := strings.Split(uri, "/")
	var err error
	for _, id := range parts {
		if len(id) == 0 {
			continue
		}
		glog.V(6).Infof("server: resolving to controller for resource id '%v'", id)
		c, err = c.Resource(id)
		if err != nil {
			http.Error(w, fmt.Sprintf("resource %v not found", id), http.StatusNotFound)
			return
		}
	}
	handler, err := c.Handle(r.Method)
	if err != nil {
		http.Error(w, fmt.Sprintf("method %v not allowed (%v)", r.Method, err), http.StatusMethodNotAllowed)
		return
	}
	glog.V(6).Infof("server: calling controller handler on body")
	// on return we close the request Body so we assume it is read synchronously
	response, err := handler(r.Body)
	var resp []byte
	if response != nil {
		resp, err = ioutil.ReadAll(response)
	}
	glog.V(6).Infof("server: called controller handler on body, response:  %v", string(resp))
	if err != nil {
		http.Error(w, fmt.Sprintf("handler error: %v", err), http.StatusBadRequest)
		return
	}
	http.ServeContent(w, r, "", time.Now(), response)
}
