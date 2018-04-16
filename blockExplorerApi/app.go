package main

//@NOTE SHYFT main func for api, sets up router and spins up a server
//to run server 'go run blockExplorerApi/*.go'
import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
)

func main() {

	router := NewRouter()
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}), handlers.AllowedOrigins([]string{"*"}))(router)))
}
