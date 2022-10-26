package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/operationspark/shorty/handlers"
	"github.com/operationspark/shorty/mongodb"
	"github.com/operationspark/shorty/shorty"
	"github.com/operationspark/shorty/testutil"
)

var mongoURI = "mongodb://localhost:27017/url-shortener-test"

func TestPOSTLink(t *testing.T) {
	t.Run("returns the Shorty by code", func(t *testing.T) {
		ogURL := "https://operationspark.org"
		reqBody := bytes.NewReader([]byte(fmt.Sprintf(`{"originalUrl":%q}`, ogURL)))

		request, _ := http.NewRequest(http.MethodPost, "/api/links", reqBody)
		response := httptest.NewRecorder()

		store, err := mongodb.NewStore(mongodb.StoreOpts{URI: mongoURI})
		if err != nil {
			t.Fatal(err)
		}

		mustDropDB(t, store)

		handlers.NewService(store).ServeHTTP(response, request)
		var got shorty.Link
		d := json.NewDecoder(response.Body)
		d.Decode(&got)

		testutil.AssertEqual(t, got.OriginalUrl, ogURL)
		testutil.AssertEqual(t, len(got.Code), 10)
		wantShortURL := fmt.Sprintf("https://ospk.org/%s", got.Code)
		testutil.AssertEqual(t, got.ShortURL, wantShortURL)

	})
}

func mustDropDB(t *testing.T, store *mongodb.MongoShortyStore) {
	err := store.Client.Database(store.DBName).Drop(context.Background())
	if err != nil {
		t.Fatalf("dropDatabase:%v", err)
	}
}
