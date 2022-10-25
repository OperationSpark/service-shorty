package shorty

import (
	"net/http"
)

var mux = newMux()

// TODO: Swap for Mongo Store
var inMemStore = NewInMemoryShortyStore()

func MainHandler(w http.ResponseWriter, r *http.Request) {
	mux.ServeHTTP(w, r)
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/urls", NewShortyServer(inMemStore).ServeHTTP)
	mux.HandleFunc("/", Resolver)

	return mux
}
