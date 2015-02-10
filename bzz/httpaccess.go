/*
A simple http server interface to Swarm
*/
package bzz

import (
	"github.com/ethereum/go-ethereum/ethutil"
	"net/http"
	"regexp"
	"time"
)

const (
	port = ":8500"
)

var (
	uriMatcher = regexp.MustCompile("^/raw/[0-9A-Fa-f]{64}$")
)

func handler(w http.ResponseWriter, r *http.Request, dpa *DPA) {
	uri := r.RequestURI
	switch {
	case r.Method == "PUT":
	case r.Method == "GET":
		if uriMatcher.MatchString(uri) {
			name := uri[5:]
			key := ethutil.Hex2Bytes(name)
			http.ServeContent(w, r, name+".bin", time.Unix(0, 0), dpa.Retrieve(key))
		} else {
			http.Error(w, "Object "+uri+" not found.", http.StatusNotFound)
		}
	default:
		http.Error(w, "Method "+r.Method+" is not supported.", http.StatusBadRequest)
	}
}

func StartHttpServer(dpa *DPA) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, dpa)
	})
	http.ListenAndServe(port, nil)
}
