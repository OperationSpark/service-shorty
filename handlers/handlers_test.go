package handlers

import (
	"testing"

	"github.com/operationspark/shorty/shorty"
	"github.com/operationspark/shorty/testutil"
)

func TestParseLinkCode(t *testing.T) {
	t.Run("grabs the Link code from the URL", func(t *testing.T) {
		tests := []struct {
			url  string
			want shorty.ShortCodeData
		}{
			{"/api/urls", shorty.ShortCodeData{Code: "", Tag: ""}},
			{"/api/urls/", shorty.ShortCodeData{Code: "", Tag: ""}},
			{"/api/urls/abc123", shorty.ShortCodeData{Code: "abc123", Tag: ""}},
			{"/api/urls/abc123/", shorty.ShortCodeData{Code: "abc123", Tag: ""}},
			{"/api/urls/abc123/t-def456", shorty.ShortCodeData{Code: "abc123", Tag: "t-def456"}},
		}

		for _, c := range tests {
			got := parseLinkCode(c.url)
			testutil.AssertEqual(t, got.Code, c.want.Code)
			testutil.AssertEqual(t, got.Tag, c.want.Tag)
		}
	})
}
