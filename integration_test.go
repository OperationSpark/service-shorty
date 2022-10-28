package function

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/operationspark/shorty/handlers"
	"github.com/operationspark/shorty/mongodb"
	"github.com/operationspark/shorty/shorty"
	"github.com/operationspark/shorty/testutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// The MongoDB client for this tests are setup in
// "integration-local-mongo_test.go" if the build tags do not include "integration",
// or "integration-dockertest_test.go", if the build tags include "integration".

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

	t.Run("uses 'customCode' if provided", func(t *testing.T) {
		reqBody := strings.NewReader(`{"customCode": "abc", "originalUrl": "https://example.com" }`)
		request, _ := http.NewRequest(http.MethodPost, "/api/urls", reqBody)
		response := httptest.NewRecorder()

		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		handlers.NewMux(store).ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusCreated)
		testutil.AssertContains(t, response.Body.String(), "https://ospk.org/abc")
	})

	t.Run("responds with 409 if code is not available", func(t *testing.T) {
		reqBody := `{"customCode": "123", "originalUrl": "https://example.com" }`
		firstReq, _ := http.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(reqBody))
		secondReq, _ := http.NewRequest(http.MethodPost, "/api/urls", strings.NewReader(reqBody))
		firstResp := httptest.NewRecorder()
		secondResp := httptest.NewRecorder()

		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		handlers.NewMux(store).ServeHTTP(firstResp, firstReq)
		handlers.NewMux(store).ServeHTTP(secondResp, secondReq)

		testutil.AssertStatus(t, secondResp.Code, http.StatusConflict)
		testutil.AssertContains(t, secondResp.Body.String(), `code: "123" already in use`)
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

func TestDELETELinksIntegration(t *testing.T) {
	t.Run("deletes a link by code", func(t *testing.T) {
		code := "abcdefg"
		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}

		seedData := shorty.Link{Code: code}
		collection := store.Client.Database(store.DBName).Collection(store.LinksCollName)

		collection.InsertOne(context.Background(), seedData)
		res1 := collection.FindOne(context.Background(), bson.D{{"code", code}})
		if res1.Err() != nil {
			t.Fatal(res1.Err())
		}

		server := handlers.NewMux(store)
		request, _ := http.NewRequest(http.MethodDelete, "/api/urls/"+code, nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		res2 := collection.FindOne(context.Background(), bson.D{{"code", code}})
		testutil.AssertEqual(t, res2.Err(), mongo.ErrNoDocuments)
		// Delete count
		testutil.AssertContains(t, response.Body.String(), "1")

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
