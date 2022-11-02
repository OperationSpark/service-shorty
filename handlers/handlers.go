package handlers

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/errorreporting"
	"github.com/operationspark/shorty/shorty"
)

type (
	LinkStore interface {
		SaveLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error)
		FindLink(ctx context.Context, code string) (shorty.Link, error)
		FindAllLinks(ctx context.Context) (shorty.Links, error)
		UpdateLink(ctx context.Context, code string, toUpdate shorty.Link) (shorty.Link, error)
		DeleteLink(ctx context.Context, code string) (int, error)
		CheckCodeInUse(ctx context.Context, code string) (bool, error)
		IncrementTotalClicks(ctx context.Context, code string) (int, error)
	}

	ShortyService struct {
		store LinkStore
		// Base service URL. Defaults to https://ospk.org
		baseURL     string
		serviceName string
		apiKey      string
		errorClient *errorreporting.Client
	}

	ServiceConfig struct {
		Store       LinkStore
		BaseURL     string
		APIkey      string
		ErrorClient *errorreporting.Client
	}
)

func NewAPIService(c ServiceConfig) *ShortyService {
	_baseURL := "https://ospk.org"
	if len(c.BaseURL) > 0 {
		_baseURL = c.BaseURL
	}

	_apiKey := os.Getenv("API_KEY")
	if len(c.APIkey) > 0 {
		_apiKey = c.APIkey
	}

	return &ShortyService{
		store:       c.Store,
		baseURL:     strings.TrimSuffix(_baseURL, "/"),
		serviceName: "system",
		apiKey:      _apiKey,
		errorClient: c.ErrorClient,
	}
}

func NewServer(apiService *ShortyService) *http.ServeMux {
	html, err := fs.Sub(content, "html")
	if err != nil {
		apiService.logError(fmt.Errorf("create html filesystem: %v", err))
		log.Fatal("could not start")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test-logging/", apiService.testLogging)
	// Find better way to ignore trailing "/"
	mux.HandleFunc("/api/urls", apiService.verifyAuth(apiService.ServeAPI))
	mux.HandleFunc("/api/urls/", apiService.verifyAuth(apiService.ServeAPI))

	mux.Handle("/favicon.ico", http.FileServer(http.FS(html)))
	mux.HandleFunc("/", apiService.ServeResolver)

	return mux
}
func (s *ShortyService) BaseURL() string {
	return s.baseURL
}

func (s *ShortyService) verifyAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("key") == s.apiKey {
			h.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
	}
}

func (s *ShortyService) logError(err error) {
	s.errorClient.Report(errorreporting.Entry{
		Error: err,
	})
	log.Print(err)
}

func (s *ShortyService) testLogging(w http.ResponseWriter, r *http.Request) {
	s.logError(fmt.Errorf("log test request:\n-> %s %s", r.Method, r.URL.Path))
	s.renderNotFound(w, r)
}

func (s *ShortyService) ServeAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

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
			s.renderNotFound(w, r)
			return
		}
		s.renderServerError(w, r, "Could not resolve link")
		s.logError(fmt.Errorf("findLink: %v", err))
	}

	_, err = s.store.IncrementTotalClicks(r.Context(), code)
	if err != nil {
		// Redirect even if there is an error. Client should not suffer if the clicks can't be updated.
		fmt.Fprintf(os.Stderr, "could not update TotalClick count: %v", err)
	}
	http.Redirect(w, r, link.OriginalUrl, http.StatusPermanentRedirect)
}

func (s *ShortyService) createLink(w http.ResponseWriter, r *http.Request) {
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
			s.logError(fmt.Errorf("checkCodeInUse: %v", err))
			http.Error(w, "could not check code", http.StatusInternalServerError)
			return
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
		s.logError(fmt.Errorf("createLink: SaveLink: %v", err))
		http.Error(w, "Problem creating short link", http.StatusInternalServerError)
		return
	}

	// Send new link JSON
	w.WriteHeader(http.StatusCreated)
	if err = newLink.ToJSON(w); err != nil {
		s.logError(fmt.Errorf("createLink: toJSON: %v", err))
		http.Error(w, "Problem marshaling your short link", http.StatusInternalServerError)
		return
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
		s.logError(fmt.Errorf("getLinks: FindLink: %v", err))
		http.Error(
			w,
			fmt.Sprintf("Could not retrieve link: %q\n", code),
			http.StatusInternalServerError,
		)
		return
	}
	link.ToJSON(w)
}

func (s *ShortyService) getLinks(w http.ResponseWriter, r *http.Request) {
	links, err := s.store.FindAllLinks(r.Context())
	if err != nil {
		s.logError(fmt.Errorf("getLinks: FindAllLinks: %v", err))
		http.Error(w, "Could not retrieve links", http.StatusInternalServerError)
		return
	}

	if err = links.ToJSON(w); err != nil {
		s.logError(fmt.Errorf("getLinks: ToJSON: %v", err))
		http.Error(w, "Problem marshaling your links", http.StatusInternalServerError)
		return
	}
}

func (s *ShortyService) updateLink(w http.ResponseWriter, r *http.Request) {
	var link shorty.Link
	err := link.FromJSON(r.Body)
	if err != nil {
		s.logError(fmt.Errorf("fromJSON: %v", err))
		http.Error(w, shorty.ErrJSONUnmarshal.Error(), http.StatusBadRequest)
		return
	}

	if len(link.CustomCode) > 0 {
		isUsed, err := s.store.CheckCodeInUse(r.Context(), link.CustomCode)
		if err != nil {
			s.logError(fmt.Errorf("checkCodeInUse: %v", err))
			http.Error(w, "Could not check customCode", http.StatusInternalServerError)
			return
		}
		if isUsed {
			http.Error(w, shorty.ErrCodeInUse.Error(), http.StatusConflict)
			return
		}
		link.GenCode(s.baseURL)
	}
	code := parseLinkCode(r.URL.Path)
	updated, err := s.store.UpdateLink(r.Context(), code, link)
	if err != nil {
		if err == shorty.ErrLinkNotFound {
			http.Error(w, shorty.ErrLinkNotFound.Error(), http.StatusNotFound)
			return
		}
		s.logError(fmt.Errorf("updateLink: %v", err))
		http.Error(w, "Could not update link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = updated.ToJSON(w)
	if err != nil {
		s.logError(fmt.Errorf("toJSON: %v", err))
		http.Error(w, "Problem marshaling link", http.StatusInternalServerError)
		return
	}
}

func (s *ShortyService) deleteLink(w http.ResponseWriter, r *http.Request) {
	code := parseLinkCode(r.URL.Path)
	count, err := s.store.DeleteLink(r.Context(), code)
	if err != nil {
		s.logError(fmt.Errorf("deleteLink: %v", err))
		http.Error(w, "Could not delete link", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, count)
}

func parseLinkCode(URLPath string) string {
	return strings.ReplaceAll(strings.TrimPrefix(URLPath, "/api/urls"), "/", "")
}
