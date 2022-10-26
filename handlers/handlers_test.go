package handlers

import (
	"testing"

	"github.com/operationspark/shorty/testutil"
)

func TestParseLinkCode(t *testing.T) {
	t.Run("grabs the Link code from the URL", func(t *testing.T) {
		tests := []struct {
			url  string
			want string
		}{
			{"/api/urls", ""},
			{"/api/urls/", ""},
			{"/api/urls/abc123", "abc123"},
			{"/api/urls/abc123/", "abc123"},
		}

		for _, c := range tests {
			got := parseLinkCode(c.url)
			testutil.AssertEqual(t, got, c.want)
		}
	})
}
