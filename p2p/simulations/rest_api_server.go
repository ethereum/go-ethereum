package simulations

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
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
	fd, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error(fmt.Sprintf("Can't listen on :%s: %v", port, err))
		return
	}
	go http.Serve(fd, serveMux)
	log.Info(fmt.Sprintf("Swarm Network Controller HTTP server started on localhost:%s", port))
}

func handle(w http.ResponseWriter, r *http.Request, c Controller) {
	requestURL := r.URL
	log.Debug(fmt.Sprintf("HTTP %s request URL: '%s', Host: '%s', Path: '%s', Referer: '%s', Accept: '%s'", r.Method, r.RequestURI, requestURL.Host, requestURL.Path, r.Referer(), r.Header.Get("Accept")))
	uri := requestURL.Path
	w.Header().Set("Content-Type", "text/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	defer r.Body.Close()
	parts := strings.Split(uri, "/")
	var err error
	for _, id := range parts {
		if len(id) == 0 {
			continue
		}
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
	// on return we close the request Body so we assume it is read synchronously
	response, err := handler(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("handler error: %v", err), http.StatusBadRequest)
		return
	}
	http.ServeContent(w, r, "", time.Now(), response)
}
