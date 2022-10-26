package shorty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type (
	ShortLink struct {
		// Shortened URL result. Ex: https://ospk.org/bas12d21dc.
		ShortURL string `json:"shortUrl"`
		// Short Code used as the path of the short URL. Ex: bas12d21dc.
		Code string `json:"code"`
		// Optional custom short code passed when creating or updating the short URL.
		CustomCode string `json:"customCode"`
		// The URL where the short URL redirects.
		OriginalUrl string `json:"originalUrl"`
		// Count of times the short URL has been used.
		TotalClicks int `json:"totalClicks"`
		// Identifier of the entity that created the short URL.
		CreatedBy string `json:"createdBy"`
		// DateTime the URL was created.
		CreatedAt time.Time `json:"createdAt"`
		// DateTime the URL was last updated.
		UpdatedAt time.Time `json:"updatedAt"`
	}

	ShortyStore interface {
		BaseURL() string
		CreateLink(ctx context.Context, newLink ShortLink) (ShortLink, error)
		GetLink(ctx context.Context, code string) (ShortLink, error)
		GetLinks(ctx context.Context) ([]ShortLink, error)
		UpdateLink(ctx context.Context, code string) (ShortLink, error)
		DeleteLink(ctx context.Context, code string) (int, error)
	}

	ShortyServer struct {
		store ShortyStore
	}
)

func NewShortyServer(s ShortyStore) *ShortyServer {
	return &ShortyServer{s}
}

func (s *ShortyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)

	switch r.Method {
	// CREATE
	case http.MethodPost:
		d := json.NewDecoder(r.Body)
		var body ShortLink
		err := d.Decode(&body)
		if err != nil {
			msg := "could not decode JSON body\n"
			http.Error(w, msg, http.StatusBadRequest)
			panic("could not decode JSON body\n" + err.Error())
		}

		newLink, err := s.store.CreateLink(r.Context(), body)
		if err != nil {
			http.Error(w, "The was a problem creating your short link", http.StatusInternalServerError)
			panic("createLink:\n" + err.Error())
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		err = encoder.Encode(newLink)
		if err != nil {
			http.Error(w, "Your link was probably created, however, there was a problem responding to your request", http.StatusInternalServerError)
			panic("encode: \n" + err.Error())
		}

		// READ
	case http.MethodGet:
		code := strings.TrimPrefix(r.URL.Path, "/api/urls/")
		if len(code) == 0 {
			links, err := s.store.GetLinks(r.Context())
			if err != nil {
				http.Error(w, "Problem retrieving links", http.StatusInternalServerError)
				panic("getLinks: \n" + err.Error())
			}

			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			err = encoder.Encode(links)
			if err != nil {
				http.Error(w, "Problem encoding links", http.StatusInternalServerError)
				panic("encode: " + err.Error())
			}
			return
		}

		link, err := s.store.GetLink(r.Context(), code)
		if err != nil {
			http.Error(w, "Problem retrieving link", http.StatusInternalServerError)
			panic("getLink: \n" + err.Error())
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		err = encoder.Encode(link)
		if err != nil {
			http.Error(w, "Problem encoding link", http.StatusInternalServerError)
			panic("encode: " + err.Error())
		}
		return

		// UPDATE
	case http.MethodPut:
		code := strings.TrimPrefix(r.URL.Path, "/api/urls/")
		if len(code) == 0 {
			http.Error(w, "Link code required", http.StatusBadRequest)
			return
		}

		updatedLink, err := s.store.UpdateLink(r.Context(), code)
		if err != nil {
			http.Error(w, "Problem updating link", http.StatusInternalServerError)
			panic("updateLink: \n" + err.Error())
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		err = encoder.Encode(updatedLink)
		if err != nil {
			http.Error(w, "Problem encoding link", http.StatusInternalServerError)
			panic("encode: " + err.Error())
		}
		return

		// DELETE
	case http.MethodDelete:
		code := strings.TrimPrefix(r.URL.Path, "/api/urls/")
		if len(code) == 0 {
			http.Error(w, "Link code required", http.StatusBadRequest)
			return
		}

		delCount, err := s.store.DeleteLink(r.Context(), code)
		if err != nil {
			http.Error(w, "Problem retrieving link", http.StatusInternalServerError)
			panic("deleteLink: \n" + err.Error())
		}

		fmt.Fprint(w, delCount)
		return
	}

}
