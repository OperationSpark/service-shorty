package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/not-found.html
var content embed.FS

type templateData struct {
	Code  string
	Title string
}

func (s *ShortyService) renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.ParseFS(content, "templates/not-found.html")
	if err != nil {
		panic(fmt.Errorf("unable to load template: %v", err))
	}

	code := parseLinkCode(r.URL.Path)
	err = t.Execute(w, templateData{
		Code:  code,
		Title: s.serviceName,
	})
	if err != nil {
		panic(fmt.Errorf("unable to render template: %v", err))
	}
}
