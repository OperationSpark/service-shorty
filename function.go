package function

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/errorreporting"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/operationspark/shorty/handlers"
	"github.com/operationspark/shorty/inmem"
	"github.com/operationspark/shorty/mongodb"
)

func init() {
	// Register an HTTP function with the Functions Framework
	// This handler name maps to the entry point name in the Google Cloud Function platform.
	// https://cloud.google.com/functions/docs/writing/write-http-functions
	functions.HTTP("ServeShorty", NewApp().ServeHTTP)
}

var store handlers.LinkStore
var errorClient *errorreporting.Client

func NewApp() *http.ServeMux {
	// Avoid variable shadow for errorClient
	var err error
	errorClient, err = initErrorReporting()
	if err != nil {
		log.Fatalf("initErrorReporting: %v", err)
	}

	store, err := initStore()
	if err != nil {
		errorClient.Report(errorreporting.Entry{
			Error: fmt.Errorf("initStore: %v", err),
		})
		log.Fatal("Could not start")
	}

	baseURL := os.Getenv("HOST_BASE_URL")
	apiKey := os.Getenv("API_KEY")

	service := handlers.NewAPIService(handlers.ServiceConfig{
		Store:       store,
		BaseURL:     baseURL,
		APIkey:      apiKey,
		ErrorClient: errorClient,
	})
	return handlers.NewServer(service)
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

func initErrorReporting() (*errorreporting.Client, error) {
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	errorClient, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName: "url-shortener",
		OnError: func(err error) {
			log.Printf("Could not log error: %v", err)
		},
	})
	if err != nil {
		return nil, err
	}
	return errorClient, nil
}

func logAndPrintError(err error) {
	errorClient.Report(errorreporting.Entry{
		Error: err,
	})
	log.Print(err)
}
