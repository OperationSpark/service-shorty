package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/operationspark/shorty/shorty"
)

type (
	LinkStore interface {
		SaveLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error)
		FindLink(ctx context.Context, code string) (shorty.Link, error)
		FindAllLinks(ctx context.Context) (shorty.Links, error)
		UpdateLink(ctx context.Context, link shorty.Link) (shorty.Link, error)
		DeleteLink(ctx context.Context, code string) (int, error)
		CheckCodeInUse(ctx context.Context, code string) (bool, error)
		IncrementTotalClicks(ctx context.Context, code string) (int, error)
	}

	ShortyService struct {
		store LinkStore
		// Base service URL. Defaults to https://ospk.org
		baseURL     string
		serviceName string
	}
)

func NewAPIService(store LinkStore, baseURL string) *ShortyService {
	_baseURL := "https://ospk.org"
	if len(baseURL) > 0 {
		_baseURL = baseURL
	}

	return &ShortyService{
		store:       store,
		baseURL:     _baseURL,
		serviceName: "system",
	}
}

func NewMux(store LinkStore) *http.ServeMux {
	baseURL := os.Getenv("HOST_BASE_URL")
	service := NewAPIService(store, baseURL)
	mux := http.NewServeMux()
	// Find better way to ignore trailing "/"
	mux.HandleFunc("/api/urls", service.ServeAPI)
	mux.HandleFunc("/api/urls/", service.ServeAPI)
	mux.HandleFunc("/", service.ServeResolver)

	return mux
}
func (s *ShortyService) BaseURL() string {
	return s.baseURL
}

func (s *ShortyService) ServeAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodPost:
		s.createLink(w, r)
		return

	case http.MethodGet:
		code := parseLinkCode(r.URL.Path)
		if len(code) == 0 {
			s.getLinks(w, r)
			return
		}
		s.getLink(w, r)
		return

	case http.MethodPut:
		s.updateLink(w, r)
		return

	case http.MethodDelete:
		s.deleteLink(w, r)
	}
}

func (s *ShortyService) ServeResolver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are accepted\n", http.StatusMethodNotAllowed)
		return
	}

	code := parseLinkCode(r.URL.Path)
	link, err := s.store.FindLink(r.Context(), code)
	if err != nil {
		if err == shorty.ErrLinkNotFound {
			http.Error(w, fmt.Sprintf("Not Found. Code: %q", code), http.StatusNotFound)
			return
		}
		http.Error(w, "Could not resolve link", http.StatusInternalServerError)
		panic(fmt.Errorf("findLink: %v", err))
	}

	_, err = s.store.IncrementTotalClicks(r.Context(), code)
	if err != nil {
		// Redirect even if there is an error. Client should not suffer if the clicks can't be updated.
		fmt.Fprintf(os.Stderr, "could not update TotalClick count")
	}
	http.Redirect(w, r, link.OriginalUrl, http.StatusPermanentRedirect)
}

func (s *ShortyService) createLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	linkInput := shorty.Link{}
	if err := linkInput.FromJSON(r.Body); err != nil {
		http.Error(w, shorty.ErrJSONUnmarshal.Error(), http.StatusBadRequest)
		return
	}

	if len(linkInput.OriginalUrl) == 0 {
		http.Error(w, `"originalUrl" field required.`, http.StatusBadRequest)
		return
	}

	// Use CustomCode if set and available
	if len(linkInput.CustomCode) > 0 {
		codeIsUsed, err := s.store.CheckCodeInUse(r.Context(), linkInput.CustomCode)
		if err != nil {
			// This should not happen
			http.Error(w, "could not check code", http.StatusInternalServerError)
			panic(err)
		}
		if codeIsUsed {
			http.Error(w, fmt.Sprintf(`code: %q already in use.`, linkInput.CustomCode), http.StatusConflict)
			return
		}
	}
	// Create and save the short link to the DB
	linkInput.GenCode(s.BaseURL())
	linkInput.UpdatedAt = time.Now()
	linkInput.CreatedAt = time.Now()
	linkInput.CreatedBy = s.serviceName

	newLink, err := s.store.SaveLink(r.Context(), linkInput)
	if err != nil {
		http.Error(w, "Problem creating short link", http.StatusInternalServerError)
		panic(fmt.Errorf("createLink: SaveLink: %v", err))
	}

	// Send new link JSON
	w.WriteHeader(http.StatusCreated)
	if err = newLink.ToJSON(w); err != nil {
		http.Error(w, "Problem marshaling your short link", http.StatusInternalServerError)
		panic(fmt.Errorf("createLink: toJSON: %v", err))
	}
}

func (s *ShortyService) getLink(w http.ResponseWriter, r *http.Request) {
	code := parseLinkCode(r.URL.Path)
	link, err := s.store.FindLink(r.Context(), code)
	if err != nil {
		if err == shorty.ErrLinkNotFound {
			http.Error(
				w,
				fmt.Sprintf("Link not found: %q", code),
				http.StatusNotFound,
			)
			return
		}
		// Other errors
		http.Error(
			w,
			fmt.Sprintf("Could not retrieve link: %q\n", code),
			http.StatusInternalServerError,
		)
		panic(fmt.Errorf("getLinks: FindLink: %v", err))
	}
	link.ToJSON(w)
}

func (s *ShortyService) getLinks(w http.ResponseWriter, r *http.Request) {
	links, err := s.store.FindAllLinks(r.Context())
	if err != nil {
		http.Error(w, "Could not retrieve links", http.StatusInternalServerError)
		panic(fmt.Errorf("getLinks: FindAllLinks: %v", err))
	}

	if err = links.ToJSON(w); err != nil {
		http.Error(w, "Problem marshaling your links", http.StatusInternalServerError)
		panic(fmt.Errorf("getLinks: ToJSON: %v", err))
	}
}

func (s *ShortyService) updateLink(w http.ResponseWriter, r *http.Request) {
	var link shorty.Link
	err := link.FromJSON(r.Body)
	if err != nil {
		http.Error(w, shorty.ErrJSONUnmarshal.Error(), http.StatusBadRequest)
		return
	}

	updated, err := s.store.UpdateLink(r.Context(), link)
	if err != nil {
		http.Error(w, "Could not update link", http.StatusInternalServerError)
		panic(fmt.Errorf("updateLink: %v", err))
	}

	w.WriteHeader(http.StatusOK)
	err = updated.ToJSON(w)
	if err != nil {
		http.Error(w, "Problem marshaling link", http.StatusInternalServerError)
		panic(fmt.Errorf("toJSON: %v", err))
	}
}

func (s *ShortyService) deleteLink(w http.ResponseWriter, r *http.Request) {
	code := parseLinkCode(r.URL.Path)
	count, err := s.store.DeleteLink(r.Context(), code)
	if err != nil {
		http.Error(w, "Could not delete link", http.StatusInternalServerError)
		panic(fmt.Errorf("deleteLink: %v", err))
	}
	fmt.Fprint(w, count)
}

func parseLinkCode(URLPath string) string {
	return strings.ReplaceAll(strings.TrimPrefix(URLPath, "/api/urls"), "/", "")
}
