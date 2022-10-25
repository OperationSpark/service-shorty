package shorty

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/operationspark/shorty/testutil"
)

func TestGETLink(t *testing.T) {
	t.Run("returns the Shorty by code", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/links", nil)
		response := httptest.NewRecorder()

		NewShortyServer(NewInMemoryShortyStore()).ServeHTTP(response, request)

		got := response.Body.String()
		want := `{}`

		testutil.AssertEqual(t, got, want)
	})
}
