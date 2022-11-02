package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"cloud.google.com/go/errorreporting"
)

//go:embed html
var content embed.FS

type (
	notFoundTemplateData struct {
		Code  string
		Title string
	}

	errorTemplateData struct {
		Error       string
		Description string
	}
)

// RenderServerError renders and responds a 404 Not Found page for the client.
func (s *ShortyService) renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	t, err := template.ParseFS(content, "html/not-found.html")
	if err != nil {
		s.errorClient.Report(errorreporting.Entry{Error: fmt.Errorf("unable to load template: %v", err)})
		s.renderServerError(w, r, "")
		return
	}

	code := parseLinkCode(r.URL.Path)
	err = t.Execute(w, notFoundTemplateData{
		Code:  code,
		Title: s.serviceName,
	})
	if err != nil {
		s.errorClient.Report(errorreporting.Entry{Error: fmt.Errorf("unable to render template: %v", err)})
		s.renderServerError(w, r, "")
		return
	}
}

// RenderServerError renders and responds with a generic error page for the client.
func (s *ShortyService) renderServerError(w http.ResponseWriter, r *http.Request, errMessage string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	t, err := template.ParseFS(content, "html/server-error.html")
	if err != nil {
		// This should never happen
		s.errorClient.Report(errorreporting.Entry{Error: fmt.Errorf("unable to load template: %v", err)})
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	description := "Sorry about that."
	if len(errMessage) > 0 {
		description = errMessage
	}
	err = t.Execute(w, errorTemplateData{
		Error:       "Ahh! Something broke.",
		Description: description,
	})
	if err != nil {
		// This should never happen
		s.errorClient.Report(errorreporting.Entry{Error: fmt.Errorf("unable to render template: %v", err)})
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}
