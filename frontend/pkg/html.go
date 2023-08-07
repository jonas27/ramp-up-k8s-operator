package pkg

import (
	"compress/gzip"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"regexp"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"golang.org/x/exp/slog"
)

// templates holds the static web server content.
//
//go:embed templates/*
var templates embed.FS

// html expose a minimalistic html page.
func (s *Server) html() func(http.ResponseWriter, uint64) {
	// cache template
	page, err := s.addHTMLTemplate("index.html")
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, characters uint64) {
		if s.Debug {
			if page, err = s.addHTMLTemplate("index.html"); err != nil {
				s.Log.Error("can't load html page", "error", err)
			}
			s.Log.Warn("reloading static conent. Should not be used in production")
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		gz := gzip.NewWriter(w)
		defer gz.Close()

		data := struct {
			Title      string
			Characters uint64
		}{"Character counter", characters}

		if err := page.Execute(gz, data); err != nil {
			slog.Error("Error executing html template", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *Server) addHTMLTemplate(name string) (*template.Template, error) {
	name = path.Join(s.TemplatePath, name)

	var bytes []byte
	var err error

	if s.Debug {
		bytes, err = os.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("error reading file: %s and error %w", name, err)
		}
	} else {
		bytes, err = templates.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("error reading file: %s and error %w", name, err)
		}
	}

	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	m.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)

	bytes, err = m.Bytes("text/html", bytes)
	if err != nil {
		return nil, fmt.Errorf("error minifying file: %s with content: %s and error: %w", name, string(bytes), err)
	}

	tmpl := template.New(name)
	tmpl, err = tmpl.Parse(string(bytes))
	if err != nil {
		return nil, fmt.Errorf("error parsing file: %s with content: %s and error: %w", name, string(bytes), err)
	}

	return tmpl, nil
}
