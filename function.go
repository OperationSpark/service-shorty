package function

import (
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/operationspark/shorty/handlers"
	"github.com/operationspark/shorty/mongodb"
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

	mongoStore, err := mongodb.NewStore(mongodb.StoreOpts{URI: mongoURI})
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/urls", handlers.NewService(mongoStore).ServeHTTP)
	mux.HandleFunc("/", handlers.Resolver)

	return mux
}
