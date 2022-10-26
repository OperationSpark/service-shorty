package shorty

import (
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	// Register an HTTP function with the Functions Framework
	// This handler name maps to the entry point name in the Google Cloud Function platform.
	// https://cloud.google.com/functions/docs/writing/write-http-functions
	functions.HTTP("ServeShorty", NewMux().ServeHTTP)
}

func NewMux() *http.ServeMux {
	mongoURI := os.Getenv("MONGO_URI")
	if len(mongoURI) == 0 {
		mongoURI = "mongodb://localhost:27017/url-shortener"
	}

	mongoStore, err := NewMongoShortyStore(MongoStoreOpts{mongoURI})
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/urls", NewShortyServer(mongoStore).ServeHTTP)
	mux.HandleFunc("/", Resolver)

	return mux
}
