package pkg

import (
	"net/http"

	"golang.org/x/exp/slog"
)

// Server holds configurable parameters.
type Server struct {
	Debug        bool
	Log          *slog.Logger
	Server       *http.Server
	TemplatePath string
}

func (s *Server) Routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.root())
	s.Server.Handler = mux
}

func (s *Server) root() http.HandlerFunc {
	html := s.html()
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			html(w, 0)
		case http.MethodPost:
			defer r.Body.Close()
			if err := r.ParseMultipartForm(0); err != nil {
				s.Log.Error("Error while parsing post multipart form", "error", err)
			}
			_ = r.FormValue("text") // implement via proto
			html(w, 0)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
