package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestAuthorization(t *testing.T) {
	t.Run("responds with 403 if 'key' header is missing", func(t *testing.T) {
		store := &mongodb.Store{Client: dbClient, DBName: dbName, LinksCollName: urlCollName}

		request, _ := http.NewRequest(http.MethodGet, "/api/urls", nil)
		response := httptest.NewRecorder()

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		handlers.NewServer(service).ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusUnauthorized)
	})

	t.Run("responds with 403 if 'key' header is incorrect", func(t *testing.T) {
		store := &mongodb.Store{Client: dbClient, DBName: dbName, LinksCollName: urlCollName}

		request, _ := http.NewRequest(http.MethodGet, "/api/urls", nil)
		request.Header.Add("key", "not-the-key")
		response := httptest.NewRecorder()

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		handlers.NewServer(service).ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusUnauthorized)
	})

	t.Run("responds with 200 if 'key' header is valid", func(t *testing.T) {
		store := &mongodb.Store{Client: dbClient, DBName: dbName, LinksCollName: urlCollName}

		request, _ := http.NewRequest(http.MethodGet, "/api/urls", nil)
		request.Header.Add("key", "test-api-key")
		response := httptest.NewRecorder()

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		handlers.NewServer(service).ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusOK)
	})
}

func TestPOSTLinkIntegration(t *testing.T) {
	t.Run("returns the Shorty by code", func(t *testing.T) {
		ogURL := "https://operationspark.org"
		reqBody := strings.NewReader(fmt.Sprintf(`{"originalUrl":%q}`, ogURL))

		request := NewRequestWithAPIKey(http.MethodPost, "/api/urls/", reqBody)
		response := httptest.NewRecorder()

		store := &mongodb.Store{Client: dbClient, DBName: dbName, LinksCollName: urlCollName}

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		handlers.NewServer(service).ServeHTTP(response, request)

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
		request := NewRequestWithAPIKey(http.MethodPost, "/api/urls/", reqBody)
		response := httptest.NewRecorder()

		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		handlers.NewServer(service).ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusBadRequest)
	})

	t.Run("uses 'customCode' if provided", func(t *testing.T) {
		reqBody := strings.NewReader(`{"customCode": "abc", "originalUrl": "https://example.com" }`)
		request := NewRequestWithAPIKey(http.MethodPost, "/api/urls/", reqBody)
		response := httptest.NewRecorder()

		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		handlers.NewServer(service).ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusCreated)
		testutil.AssertContains(t, response.Body.String(), "https://ospk.org/abc")
	})

	t.Run("responds with 409 if code is not available", func(t *testing.T) {
		reqBody := `{"customCode": "123", "originalUrl": "https://example.com" }`
		firstReq := NewRequestWithAPIKey(http.MethodPost, "/api/urls/", strings.NewReader(reqBody))
		secondReq := NewRequestWithAPIKey(http.MethodPost, "/api/urls/", strings.NewReader(reqBody))
		firstResp := httptest.NewRecorder()
		secondResp := httptest.NewRecorder()

		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		handlers.NewServer(service).ServeHTTP(firstResp, firstReq)
		handlers.NewServer(service).ServeHTTP(secondResp, secondReq)

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

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		server := handlers.NewServer(service)

		wantContained := `"code":"abc1234"`

		request := NewRequestWithAPIKey(http.MethodGet, "/api/urls/", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusOK)
		testutil.AssertContains(t, response.Body.String(), wantContained)
		testutil.AssertContains(t, response.Header().Get("Content-Type"), "application/json")
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

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		server := handlers.NewServer(service)
		request := NewRequestWithAPIKey(http.MethodDelete, "/api/urls/"+code, nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		res2 := collection.FindOne(context.Background(), bson.D{{"code", code}})
		testutil.AssertEqual(t, res2.Err(), mongo.ErrNoDocuments)
		// Delete count
		testutil.AssertContains(t, response.Body.String(), "1")

	})
}

func TestUPDATELinksIntegration(t *testing.T) {
	t.Run("updates a link's original URL by code", func(t *testing.T) {
		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		seedData := shorty.Link{
			Code:        "abcdef123",
			OriginalUrl: "https://quii.gitbook.io/learn-go-with-tests/",
		}

		newURL := "https://changelog.com/gotime/253"

		collection := store.Client.Database(store.DBName).Collection(store.LinksCollName)
		_, err := collection.InsertOne(context.Background(), seedData)
		if err != nil {
			t.Fatal(err)
		}

		var updateBody bytes.Buffer
		seedData.OriginalUrl = newURL
		seedData.ToJSON(&updateBody)

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		server := handlers.NewServer(service)
		request := NewRequestWithAPIKey(http.MethodPut, "/api/urls/"+seedData.Code, &updateBody)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusOK)
		testutil.AssertContains(t, response.Body.String(), fmt.Sprintf(`"originalUrl":%q`, newURL))

		// Check database
		res := collection.FindOne(
			context.Background(),
			bson.D{{"code", seedData.Code}},
		)
		if res.Err() != nil {
			t.Fatal(err)
		}
		var updatedLink shorty.Link
		res.Decode(&updatedLink)
		testutil.AssertEqual(t, updatedLink.OriginalUrl, newURL)
	})

	t.Run("404s if code not found", func(t *testing.T) {
		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		server := handlers.NewServer(service)
		request := NewRequestWithAPIKey(http.MethodPut, "/api/urls/notacode", strings.NewReader(`{}`))
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusNotFound)

	})

	t.Run("updates 'customCode' field", func(t *testing.T) {
		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		server := handlers.NewServer(service)
		createReq := NewRequestWithAPIKey(http.MethodPost, "/api/urls", strings.NewReader(`{"originalUrl":"https://netflix.com"}`))
		createResp := httptest.NewRecorder()

		server.ServeHTTP(createResp, createReq)
		testutil.AssertStatus(t, createResp.Code, http.StatusCreated)

		var toUpdate shorty.Link
		toUpdate.FromJSON(createResp.Body)
		updateReq := NewRequestWithAPIKey(http.MethodPut, "/api/urls/"+toUpdate.Code, strings.NewReader(`{"customCode":"nflx"}`))
		updateResp := httptest.NewRecorder()

		server.ServeHTTP(updateResp, updateReq)

		testutil.AssertStatus(t, updateResp.Code, http.StatusOK)
		// Make sure customCode, code, and shortUrl fields all updated
		testutil.AssertContains(t, updateResp.Body.String(), `"customCode":"nflx"`)
		testutil.AssertContains(t, updateResp.Body.String(), `"code":"nflx"`)
		testutil.AssertContains(t, updateResp.Body.String(), `"shortUrl":"https://ospk.org/nflx"`)
	})

	t.Run("fails if 'customCode' value already in use", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func TestCreateLinkAndRedirect(t *testing.T) {
	t.Run("creates and uses a short link", func(t *testing.T) {
		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}

		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store:  store,
			APIkey: "test-api-key",
		})
		server := handlers.NewServer(service)

		originalURL := "https://greenlight.operationspark.org/dashboard?subview=overview"
		createLinkBody := strings.NewReader(fmt.Sprintf(`{"originalUrl": %q }`, originalURL))
		createLinkReq := NewRequestWithAPIKey(http.MethodPost, "/api/urls/", createLinkBody)
		createLinkResp := httptest.NewRecorder()

		// POST to create a new short link
		server.ServeHTTP(createLinkResp, createLinkReq)

		var newLink shorty.Link
		json.NewDecoder(createLinkResp.Body).Decode(&newLink)

		// Use new short link
		useLinkReq := NewRequestWithAPIKey(http.MethodGet, "/"+newLink.Code, nil)
		redirectResp := httptest.NewRecorder()

		server.ServeHTTP(redirectResp, useLinkReq)

		testutil.AssertStatus(t, redirectResp.Code, http.StatusPermanentRedirect)
		testutil.AssertContains(t, redirectResp.Body.String(), originalURL)

		// Check click count increment
		getLinkReq := NewRequestWithAPIKey(http.MethodGet, "/api/urls/"+newLink.Code, nil)
		getLinkResp := httptest.NewRecorder()
		server.ServeHTTP(getLinkResp, getLinkReq)

		testutil.AssertContains(t, getLinkResp.Body.String(), `"totalClicks":1`)
		// UpdatedAt rounded to nearest second
		wantUpdatedAtISO := strings.TrimSuffix(time.Now().UTC().Format(time.RFC3339), "Z")
		testutil.AssertContains(
			t,
			getLinkResp.Body.String(),
			fmt.Sprintf(`"updatedAt":"%s`, wantUpdatedAtISO),
		)
		testutil.AssertStatus(t, getLinkResp.Code, http.StatusOK)
	})
}

func TestNotFoundPage(t *testing.T) {
	t.Run("renders not-found page when the no link exists for a given code", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/9234234", nil)
		response := httptest.NewRecorder()

		store := &mongodb.Store{
			Client:        dbClient,
			DBName:        dbName,
			LinksCollName: urlCollName,
		}
		service := handlers.NewAPIService(handlers.ServiceConfig{
			Store: store,
		})
		server := handlers.NewServer(service)

		server.ServeHTTP(response, request)

		html := response.Body.String()
		testutil.AssertContains(t, html, "<h2 class=\"error-message\">INVALID CODE</h2>")
		testutil.AssertContains(t, html, "<code>9234234</code>")
		testutil.AssertStatus(t, response.Code, http.StatusNotFound)
	})
}

func TestFavicon(t *testing.T) {
	t.Run("serves favicon.ico", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/favicon.ico", nil)
		response := httptest.NewRecorder()

		service := handlers.NewAPIService(handlers.ServiceConfig{})
		server := handlers.NewServer(service)

		server.ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusOK)
	})
}

func NewRequestWithAPIKey(method, path string, body io.Reader) *http.Request {
	r, _ := http.NewRequest(method, path, body)
	r.Header.Add("key", "test-api-key")
	return r
}
