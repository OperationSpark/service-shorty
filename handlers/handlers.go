package handlers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/errorreporting"
	"github.com/operationspark/shorty/gcp"
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
		AddTagActivity(ctx context.Context, codeData shorty.ShortCodeData) (int, error)
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
		apiService.logError(fmt.Errorf("create html filesystem: %v", err), "")
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

func (s *ShortyService) logError(err error, trace string) {
	s.errorClient.Report(errorreporting.Entry{
		Error: err,
	})
	log.Println(gcp.LogEntry{
		Severity:  "ERROR",
		Message:   err.Error(),
		Component: s.serviceName,
		Trace:     trace,
	})
}

// GetTrace derives the traceID associated with the current request.
func (s *ShortyService) getTrace(r *http.Request) string {
	projectID := os.Getenv("GCP_PROJECT_ID")
	var trace string
	if len(projectID) == 0 {
		return trace
	}

	traceHeader := r.Header.Get("X-Cloud-Trace-Context")
	traceParts := strings.Split(traceHeader, "/")
	if len(traceParts) > 0 && len(traceParts[0]) > 0 {
		trace = fmt.Sprintf("projects/%s/traces/%s", projectID, traceParts[0])
	}
	return trace
}

func (s *ShortyService) testLogging(w http.ResponseWriter, r *http.Request) {
	s.logError(
		fmt.Errorf("log test request:\n-> %s %s", r.Method, r.URL.Path),
		s.getTrace(r),
	)
	s.renderNotFound(w, r)
}

func (s *ShortyService) ServeAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	switch r.Method {

	case http.MethodPost:
		s.createLink(w, r)
		return

	case http.MethodGet:
		codeData := parseLinkCode(r.URL.Path)
		if len(codeData.Code) == 0 {
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

	if r.URL.Path == "/" {
		http.Redirect(w, r, "https://www.operationspark.org", http.StatusTemporaryRedirect)
		return
	}

	codeData := parseLinkCode(r.URL.Path)

	link, err := s.store.FindLink(r.Context(), codeData.Code)
	if err != nil {
		if err == shorty.ErrLinkNotFound {
			s.renderNotFound(w, r)
			return
		}
		s.renderServerError(w, r, "Could not resolve link")
		s.logError(fmt.Errorf("findLink: %v", err), s.getTrace(r))
	}

	_, err = s.store.IncrementTotalClicks(r.Context(), codeData.Code)
	if err != nil {
		// Redirect even if there is an error. Client should not suffer if the clicks can't be updated.
		fmt.Fprintf(os.Stderr, "could not update TotalClick count: %v", err)
	}

	_, err = s.store.AddTagActivity(r.Context(), codeData)

	http.Redirect(w, r, link.OriginalUrl, http.StatusTemporaryRedirect)
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

	if err := validateURL(linkInput.OriginalUrl); err != nil {
		if errors.Is(err, shorty.ErrRelativeURL) {
			http.Error(w, fmt.Sprintf("URL: %q is relative. URLs must be absolute", linkInput.OriginalUrl), http.StatusBadRequest)
			return
		}
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Use CustomCode if set and available
	if len(linkInput.CustomCode) > 0 {
		codeIsUsed, err := s.store.CheckCodeInUse(r.Context(), linkInput.CustomCode)
		if err != nil {
			// This should not happen
			s.logError(fmt.Errorf("checkCodeInUse: %v", err), s.getTrace(r))
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
		s.logError(fmt.Errorf("createLink: SaveLink: %v", err), s.getTrace(r))
		http.Error(w, "Problem creating short link", http.StatusInternalServerError)
		return
	}

	// Send new link JSON
	w.WriteHeader(http.StatusCreated)
	if err = newLink.ToJSON(w); err != nil {
		s.logError(fmt.Errorf("createLink: toJSON: %v", err), s.getTrace(r))
		http.Error(w, "Problem marshaling your short link", http.StatusInternalServerError)
		return
	}
}

func (s *ShortyService) getLink(w http.ResponseWriter, r *http.Request) {
	codeData := parseLinkCode(r.URL.Path)
	link, err := s.store.FindLink(r.Context(), codeData.Code)
	if err != nil {
		if err == shorty.ErrLinkNotFound {
			http.Error(
				w,
				fmt.Sprintf("Link not found: %q", codeData.Code),
				http.StatusNotFound,
			)
			return
		}

		// Other errors
		s.logError(fmt.Errorf("getLinks: FindLink: %v", err), s.getTrace(r))
		http.Error(
			w,
			fmt.Sprintf("Could not retrieve link: %q\n", codeData.Code),
			http.StatusInternalServerError,
		)
		return
	}
	link.ToJSON(w)
}

func (s *ShortyService) getLinks(w http.ResponseWriter, r *http.Request) {
	links, err := s.store.FindAllLinks(r.Context())
	if err != nil {
		s.logError(fmt.Errorf("getLinks: FindAllLinks: %v", err), s.getTrace(r))
		http.Error(w, "Could not retrieve links", http.StatusInternalServerError)
		return
	}

	if err = links.ToJSON(w); err != nil {
		s.logError(fmt.Errorf("getLinks: ToJSON: %v", err), s.getTrace(r))
		http.Error(w, "Problem marshaling your links", http.StatusInternalServerError)
		return
	}
}

func (s *ShortyService) updateLink(w http.ResponseWriter, r *http.Request) {
	var link shorty.Link
	err := link.FromJSON(r.Body)
	if err != nil {
		s.logError(fmt.Errorf("fromJSON: %v", err), s.getTrace(r))
		http.Error(w, shorty.ErrJSONUnmarshal.Error(), http.StatusBadRequest)
		return
	}

	if len(link.CustomCode) > 0 {
		isUsed, err := s.store.CheckCodeInUse(r.Context(), link.CustomCode)
		if err != nil {
			s.logError(fmt.Errorf("checkCodeInUse: %v", err), s.getTrace(r))
			http.Error(w, "Could not check customCode", http.StatusInternalServerError)
			return
		}
		if isUsed {
			http.Error(w, shorty.ErrCodeInUse.Error(), http.StatusConflict)
			return
		}
		link.GenCode(s.baseURL)
	}
	codeData := parseLinkCode(r.URL.Path)
	updated, err := s.store.UpdateLink(r.Context(), codeData.Code, link)
	if err != nil {
		if err == shorty.ErrLinkNotFound {
			http.Error(w, shorty.ErrLinkNotFound.Error(), http.StatusNotFound)
			return
		}
		s.logError(fmt.Errorf("updateLink: %v", err), s.getTrace(r))
		http.Error(w, "Could not update link", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = updated.ToJSON(w)
	if err != nil {
		s.logError(fmt.Errorf("toJSON: %v", err), s.getTrace(r))
		http.Error(w, "Problem marshaling link", http.StatusInternalServerError)
		return
	}
}

func (s *ShortyService) deleteLink(w http.ResponseWriter, r *http.Request) {
	codeData := parseLinkCode(r.URL.Path)
	count, err := s.store.DeleteLink(r.Context(), codeData.Code)
	if err != nil {
		s.logError(fmt.Errorf("deleteLink: %v", err), s.getTrace(r))
		http.Error(w, "Could not delete link", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, count)
}

func parseLinkCode(URLPath string) shorty.ShortCodeData {
	// parse code and tag from link path ex: /api/urls/abc123/tag
	path := strings.TrimPrefix(URLPath, "/api/urls")
	path = strings.Trim(path, "/")

	codes := strings.Split(path, "/")

	if len(codes) == 0 {
		return shorty.ShortCodeData{
			Code: "",
			Tag:  "",
		}
	}

	if len(codes) == 1 {
		return shorty.ShortCodeData{
			Code: codes[0],
			Tag:  "",
		}
	}

	return shorty.ShortCodeData{
		Code: codes[0],
		Tag:  codes[1],
	}
}

func validateURL(toShorten string) error {
	u, err := url.Parse(toShorten)
	if err != nil {
		return shorty.ErrInvalidURL
	}
	if !u.IsAbs() {
		return shorty.ErrRelativeURL
	}
	return nil
}
