package shorty

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type (
	Link struct {
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

	Links []*Link

	Tag struct {
		// Short Code used as the optional second param of the short URL path. Ex: /:shortCode/abc123
		//  /:shortCode/:tagCode
		Code string `json:"code" bson:"code"`

		// List of click/url history for the tag.
		Activity []struct {
			// DateTime the tag was created.
			CreatedAt time.Time `json:"createdAt" bson:"createdAt"`

			// Short URL code used
			ShortCode string `json:"shortCode" bson:"shortCode"`
		} `json:"activity" bson:"activity"`

		Data map[string]interface{} `json:"data" bson:"data"`

		// Identifier of the entity that created the short URL.
		CreatedBy string `json:"createdBy" bson:"createdBy"`

		// DateTime the tag was created.
		CreatedAt time.Time `json:"createdAt" bson:"createdAt"`

		// DateTime the tag was last updated.
		UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
	}

	Tags []*Tag

	ShortCodeData struct {
		Code string
		Tag  string
	}
)

// FromJSON unmarshals a request's JSON body into a Link.
func (sl *Link) FromJSON(r io.Reader) error {
	if err := json.NewDecoder(r).Decode(sl); err != nil {
		return fmt.Errorf("decode: %v", err)
	}
	return nil
}

// ToJSON marshals a Link into JSON and writes the result to a Writer.
func (sl *Link) ToJSON(w io.Writer) error {
	if err := json.NewEncoder(w).Encode(sl); err != nil {
		return fmt.Errorf("encode: %v", err)
	}
	return nil
}

// GenCode generates and sets the Code, CustomCode, and ShortURL fields on the Link.
// If the Link already has a CustomCode, Code and ShortURL will be set to that value.
func (sl *Link) GenCode(baseURL string) {
	// Check if "customCode" set and use it if so.
	if len(sl.CustomCode) > 0 {
		sl.Code = sl.CustomCode
		sl.ShortURL = fmt.Sprintf("%s/%s", baseURL, sl.CustomCode)
		return
	}

	// Generate random code if customCode not set
	code := CreateCode()
	sl.Code = code
	sl.CustomCode = code
	sl.ShortURL = fmt.Sprintf("%s/%s", baseURL, code)
}

// ToJSON marshals a list of Links into JSON and writes the result to a Writer.
func (l *Links) ToJSON(w io.Writer) error {
	if err := json.NewEncoder(w).Encode(l); err != nil {
		return fmt.Errorf("encode: %v", err)
	}
	return nil
}
