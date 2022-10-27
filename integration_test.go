//go:build integration
// +build integration

package function

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/operationspark/shorty/handlers"
	"github.com/operationspark/shorty/mongodb"
	"github.com/operationspark/shorty/shorty"
	"github.com/operationspark/shorty/testutil"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var dbClient *mongo.Client
var dbName = "url-shortener-test"
var urlCollName = "urls"

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pull mongodb docker image for version 5.0
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "5.0",
		Env: []string{
			// username and password for mongodb superuser
			"MONGO_INITDB_ROOT_USERNAME=root",
			"MONGO_INITDB_ROOT_PASSWORD=password",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		var err error
		dbClient, err = mongo.Connect(
			context.TODO(),
			options.Client().ApplyURI(
				fmt.Sprintf("mongodb://root:password@localhost:%s", resource.GetPort("27017/tcp")),
			),
		)
		if err != nil {
			return err
		}
		return dbClient.Ping(context.TODO(), nil)
	})

	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// run tests
	code := m.Run()

	// When you're done, kill and remove the container
	if err = pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	// disconnect mongodb client
	if err = dbClient.Disconnect(context.TODO()); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func TestPOSTLinkIntegration(t *testing.T) {
	t.Run("returns the Shorty by code", func(t *testing.T) {
		ogURL := "https://operationspark.org"
		reqBody := strings.NewReader(fmt.Sprintf(`{"originalUrl":%q}`, ogURL))

		request, _ := http.NewRequest(http.MethodPost, "/api/urls", reqBody)
		response := httptest.NewRecorder()

		store := &mongodb.Store{Client: dbClient, DBName: dbName, LinksCollName: urlCollName}

		handlers.NewMux(store).ServeHTTP(response, request)

		var got shorty.Link
		d := json.NewDecoder(response.Body)
		d.Decode(&got)

		testutil.AssertEqual(t, got.OriginalUrl, ogURL)
		testutil.AssertEqual(t, len(got.Code), 10)
		wantShortURL := fmt.Sprintf("https://ospk.org/%s", got.Code)
		testutil.AssertEqual(t, got.ShortURL, wantShortURL)
	})

	t.Run("errors if no 'originalUrl' field in body", func(t *testing.T) {
		reqBody := strings.NewReader(`{}`)
		request, _ := http.NewRequest(http.MethodPost, "/api/urls", reqBody)
		response := httptest.NewRecorder()

		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		handlers.NewMux(store).ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusBadRequest)
	})

	t.Run("reuses code if no 'originalUrl' field matches an existing link", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func TestGETLinksIntegration(t *testing.T) {
	t.Run("returns all the links in the store", func(t *testing.T) {
		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}

		seedData := shorty.Link{Code: "abc1234"}
		store.Client.Database(store.DBName).Collection(store.LinksCollName).InsertOne(context.Background(), seedData)

		server := handlers.NewMux(store)

		wantContained := `"code":"abc1234"`

		request, _ := http.NewRequest(http.MethodGet, "/api/urls/", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusOK)
		testutil.AssertContains(t, response.Body.String(), wantContained)

	})
}

func TestCreateLinkAndRedirect(t *testing.T) {
	t.Run("creates and uses a short link", func(t *testing.T) {
		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}

		server := handlers.NewMux(store)

		originalURL := "https://greenlight.operationspark.org/dashboard?subview=overview"
		createLinkBody := strings.NewReader(fmt.Sprintf(`{"originalUrl": %q }`, originalURL))
		createLinkReq, _ := http.NewRequest(http.MethodPost, "/api/urls/", createLinkBody)
		createLinkResp := httptest.NewRecorder()

		// POST to create a new short link
		server.ServeHTTP(createLinkResp, createLinkReq)

		var newLink shorty.Link
		json.NewDecoder(createLinkResp.Body).Decode(&newLink)

		// Use new short link
		useLinkReq, _ := http.NewRequest(http.MethodGet, "/"+newLink.Code, nil)
		redirectResp := httptest.NewRecorder()

		server.ServeHTTP(redirectResp, useLinkReq)

		testutil.AssertStatus(t, redirectResp.Code, http.StatusPermanentRedirect)
		testutil.AssertContains(t, redirectResp.Body.String(), originalURL)
	})
}
