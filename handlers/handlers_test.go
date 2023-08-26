package handlers

import (
	"testing"

	"github.com/operationspark/shorty/testutil"
)

func TestParseLinkCode(t *testing.T) {
	t.Run("grabs the Link code from the URL", func(t *testing.T) {
		tests := []struct {
			url  string
			want *ShortCodeData
		}{
			{"/api/urls", &ShortCodeData{code: "", tag: ""}},
			{"/api/urls/", &ShortCodeData{code: "", tag: ""}},
			{"/api/urls/abc123", &ShortCodeData{code: "abc123", tag: ""}},
			{"/api/urls/abc123/", &ShortCodeData{code: "abc123", tag: ""}},
			{"/api/urls/abc123/t-def456", &ShortCodeData{code: "abc123", tag: "t-def456"}},
		}

		for _, c := range tests {
			got := parseLinkCode(c.url)
			testutil.AssertEqual(t, got.code, c.want.code)
			testutil.AssertEqual(t, got.tag, c.want.tag)
		}
	})
}
