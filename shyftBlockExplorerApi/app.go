package main

//@NOTE SHYFT main func for api, sets up router and spins up a server
//to run server 'go run shyftBlockExplorerApi/*.go'
import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
)

func main() {

	router := NewRouter()
	port := "8080"
	log.Printf("Listening on port " + " " + port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}), handlers.AllowedOrigins([]string{"*"}))(router)))
}
