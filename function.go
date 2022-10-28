package function

import (
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/operationspark/shorty/handlers"
	"github.com/operationspark/shorty/inmem"
	"github.com/operationspark/shorty/mongodb"
)

func init() {
	// Register an HTTP function with the Functions Framework
	// This handler name maps to the entry point name in the Google Cloud Function platform.
	// https://cloud.google.com/functions/docs/writing/write-http-functions
	functions.HTTP("ServeShorty", NewMux().ServeHTTP)
}

var store handlers.LinkStore

func NewMux() *http.ServeMux {
	store, err := initStore()
	if err != nil {
		panic(err)
	}

	return handlers.NewMux(store)
}

// InitStore initializes the ShortyStore to either a MongoDB or an in-memory implementation.
func initStore() (handlers.LinkStore, error) {
	if os.Getenv("CI") == "true" {
		return inmem.NewStore(), nil
	}
	mongoURI := os.Getenv("MONGO_URI")
	if len(mongoURI) == 0 {
		mongoURI = "mongodb://localhost:27017/url-shortener"
	}

	store, err := mongodb.NewStore(mongodb.StoreOpts{URI: mongoURI})
	if err != nil {
		return nil, err
	}
	return store, nil
}
