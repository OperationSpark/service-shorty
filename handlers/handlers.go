package handlers

import (
	"context"
	"net/http"

	"github.com/operationspark/shorty/shortlink"
)

type (
	ShortyStore interface {
		CreateLink(ctx context.Context, newLink shortlink.ShortLink) (shortlink.ShortLink, error)
		GetLink(ctx context.Context, code string) (shortlink.ShortLink, error)
		GetLinks(ctx context.Context) ([]shortlink.ShortLink, error)
		UpdateLink(ctx context.Context, code string) (shortlink.ShortLink, error)
		DeleteLink(ctx context.Context, code string) (int, error)
	}

	ShortyService struct {
		store ShortyStore
	}

	jsonError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

func NewShortyService(store ShortyStore) *ShortyService {
	return &ShortyService{store: store}
}

func (s *ShortyService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// CREATE
	case http.MethodPost:
		s.createLink(w, r)
		return
		// READ
	case http.MethodGet:
		// UPDATE
	case http.MethodPut:
		// DELETE
	case http.MethodDelete:
	}
}

func (s *ShortyService) createLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	linkInput := shortlink.ShortLink{}
	if err := linkInput.FromJSON(r.Body); err != nil {
		http.Error(w, "Unable to parse JSON", http.StatusBadRequest)
	}

	// Create and save the short link to the DB
	newLink, err := s.store.CreateLink(r.Context(), linkInput)
	if err != nil {
		http.Error(w, "Problem creating short link", http.StatusInternalServerError)
	}

	// Send new link JSON
	if err = newLink.ToJSON(w); err != nil {
		http.Error(w, "Problem marshaling your short link", http.StatusInternalServerError)
	}
}
