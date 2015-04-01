package main

import (
	"net/http"

	"github.com/rs/cors"
	"github.com/zenazn/goji"
)

func main() {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://foo.com"},
	})
	goji.Use(c.Handler)

	goji.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"hello\": \"world\"}"))
	})

	goji.Serve()
}
