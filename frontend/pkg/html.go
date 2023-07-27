package pkg

import (
	"compress/gzip"
	"embed"
	"html/template"
	"net/http"
	"os"
	"path"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"golang.org/x/exp/slog"
)

// templates holds the static web server content.
//
//go:embed templates/*
var templates embed.FS

// rootGet expose a minimalistic html page
func (s *Server) rootGet() func(w http.ResponseWriter) {
	// cache template
	page := s.addHTMLTemplate("index.html")
	return func(w http.ResponseWriter) {
		if s.Debug {
			page = s.addHTMLTemplate("index.html")
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		gz := gzip.NewWriter(w)
		defer gz.Close()

		data := struct {
			Title string
		}{Title: "Character counter"}

		if err := page.Execute(gz, data); err != nil {
			slog.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)

		}
	}
}

func (s *Server) addHTMLTemplate(name string) *template.Template {
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	name = path.Join(s.TemplatePath, name)

	var bytes []byte
	var err error

	if s.Debug {
		bytes, err = os.ReadFile(name)
		if err != nil {
			panic(err)
		}
	} else {
		bytes, err = templates.ReadFile(name)
		if err != nil {
			panic(err)
		}
	}

	bytes, err = m.Bytes("text/html", bytes)
	if err != nil {
		panic(err)
	}

	tmpl := template.New(name)
	tmpl, err = tmpl.Parse(string(bytes))
	if err != nil {
		panic(err)
	}

	return tmpl
}
