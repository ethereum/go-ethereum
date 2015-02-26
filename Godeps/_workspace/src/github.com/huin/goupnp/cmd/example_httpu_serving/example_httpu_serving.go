package main

import (
	"log"
	"net/http"

	"github.com/huin/goupnp/httpu"
)

func main() {
	srv := httpu.Server{
		Addr:      "239.255.255.250:1900",
		Multicast: true,
		Handler: httpu.HandlerFunc(func(r *http.Request) {
			log.Printf("Got %s %s message from %v: %v", r.Method, r.URL.Path, r.RemoteAddr, r.Header)
		}),
	}
	err := srv.ListenAndServe()
	log.Printf("Serving failed with error: %v", err)
}
