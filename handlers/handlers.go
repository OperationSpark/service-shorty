package handlers

import (
	"context"
	"net/http"

	"github.com/operationspark/shorty/shorty"
)

type (
	LinkStore interface {
		CreateLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error)
		GetLink(ctx context.Context, code string) (shorty.Link, error)
		GetLinks(ctx context.Context) ([]shorty.Link, error)
		UpdateLink(ctx context.Context, code string) (shorty.Link, error)
		DeleteLink(ctx context.Context, code string) (int, error)
	}

	ShortyService struct {
		store LinkStore
	}
)

func NewService(store LinkStore) *ShortyService {
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

	linkInput := shorty.Link{}
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
