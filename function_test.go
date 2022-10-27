package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/operationspark/shorty/handlers"
	"github.com/operationspark/shorty/inmem"
	"github.com/operationspark/shorty/shorty"
	"github.com/operationspark/shorty/testutil"
)

func TestPOSTLink(t *testing.T) {
	t.Run("returns the Shorty by code", func(t *testing.T) {
		ogURL := "https://operationspark.org"
		reqBody := bytes.NewReader([]byte(fmt.Sprintf(`{"originalUrl":%q}`, ogURL)))

		request, _ := http.NewRequest(http.MethodPost, "/api/urls/", reqBody)
		response := httptest.NewRecorder()

		store := inmem.NewStore()

		handlers.NewMux(store).ServeHTTP(response, request)
		var got shorty.Link
		d := json.NewDecoder(response.Body)
		d.Decode(&got)

		testutil.AssertEqual(t, got.OriginalUrl, ogURL)
		testutil.AssertEqual(t, len(got.Code), 10)
		wantShortURL := fmt.Sprintf("https://ospk.org/%s", got.Code)
		testutil.AssertEqual(t, got.ShortURL, wantShortURL)

	})
}

func TestGETLink(t *testing.T) {
	store := inmem.NewStore()
	store.Store = map[string]shorty.Link{
		"abc123": {Code: "abc123"},
	}

	server := handlers.NewMux(store)

	tests := []struct {
		name       string
		endpoint   string
		wantBody   string
		statusCode int
	}{
		{
			name:       "GET abc123",
			endpoint:   "/abc123",
			wantBody:   `{"shortUrl":"","code":"abc123","customCode":"","originalUrl":"","totalClicks":0,"createdBy":"","createdAt":"0001-01-01T00:00:00Z","updatedAt":"0001-01-01T00:00:00Z"}` + "\n",
			statusCode: http.StatusOK,
		},
		{
			name:       "GET not existent",
			endpoint:   "/not-a-valid-code",
			wantBody:   "Link not found: \"not-a-valid-code\"\n",
			statusCode: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/urls%s", test.endpoint), nil)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			testutil.AssertStatus(t, response.Code, test.statusCode)
			testutil.AssertResponseBody(t, response.Body.String(), test.wantBody)
		})
	}
}

func TestGETLinks(t *testing.T) {
	t.Run("returns all the links in the store", func(t *testing.T) {
		store := inmem.NewStore()
		store.Store = map[string]shorty.Link{
			"abc123": {Code: "abc123"},
		}

		server := handlers.NewMux(store)

		wantBody := `[{"shortUrl":"","code":"abc123","customCode":"","originalUrl":"","totalClicks":0,"createdBy":"","createdAt":"0001-01-01T00:00:00Z","updatedAt":"0001-01-01T00:00:00Z"}]` + "\n"

		request, _ := http.NewRequest(http.MethodGet, "/api/urls/", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusOK)
		testutil.AssertResponseBody(t, response.Body.String(), wantBody)
	})

	t.Run("returns empty list if there a no links in the store", func(t *testing.T) {
		store := inmem.NewStore()
		server := handlers.NewMux(store)

		wantBody := "[]\n"

		request, _ := http.NewRequest(http.MethodGet, "/api/urls/", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		testutil.AssertStatus(t, response.Code, http.StatusOK)
		testutil.AssertResponseBody(t, response.Body.String(), wantBody)
	})
}
