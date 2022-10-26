package shortlink

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type (
	ShortLink struct {
		// Shortened URL result. Ex: https://ospk.org/bas12d21dc.
		ShortURL string `json:"shortUrl" bson:"shortUrl"`
		// Short Code used as the path of the short URL. Ex: bas12d21dc.
		Code string `json:"code" bson:"code"`
		// Optional custom short code passed when creating or updating the short URL.
		CustomCode string `json:"customCode" bson:"customCode"`
		// The URL where the short URL redirects.
		OriginalUrl string `json:"originalUrl" bson:"originalUrl"`
		// Count of times the short URL has been used.
		TotalClicks int `json:"totalClicks" bson:"totalClicks"`
		// Identifier of the entity that created the short URL.
		CreatedBy string `json:"createdBy" bson:"createdBy"`
		// DateTime the URL was created.
		CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
		// DateTime the URL was last updated.
		UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
	}

	Links []*ShortLink

	ShortyStore interface {
		CreateLink(ctx context.Context, newLink ShortLink) (ShortLink, error)
		GetLink(ctx context.Context, code string) (ShortLink, error)
		GetLinks(ctx context.Context) (Links, error)
		UpdateLink(ctx context.Context, code string) (ShortLink, error)
		DeleteLink(ctx context.Context, code string) (int, error)
	}
)

func (sl *ShortLink) FromJSON(r io.Reader) error {
	if err := json.NewDecoder(r).Decode(sl); err != nil {
		return fmt.Errorf("decode: %v", err)
	}
	return nil
}

func (sl *ShortLink) ToJSON(w io.Writer) error {
	if err := json.NewEncoder(w).Encode(sl); err != nil {
		return fmt.Errorf("encode: %v", err)
	}
	return nil
}

func (sl *ShortLink) GenCode(baseURL string) {
	code := CreateCode()

	sl.Code = code
	sl.CustomCode = code
	sl.ShortURL = fmt.Sprintf("%s/%s", baseURL, code)
}

func (l *Links) toJSON(w io.Writer) error {
	if err := json.NewEncoder(w).Encode(l); err != nil {
		return fmt.Errorf("encode: %v", err)
	}
	return nil
}
