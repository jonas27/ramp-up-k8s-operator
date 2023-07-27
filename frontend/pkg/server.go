package pkg

import (
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/exp/slog"
)

// Server holds configurable parameters
type Server struct {
	Debug        bool
	Log          *slog.Logger
	Mux          *http.ServeMux
	Server       *http.Server
	TemplatePath string
}

func (s *Server) Serve() error {
	s.Server.Handler = s.Mux
	s.routes()
	if err := s.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("the server failed with error: %w", err)
	}
	return nil
}

func (s *Server) routes() {
	s.Mux.HandleFunc("/", s.root())
}

func (s *Server) root() http.HandlerFunc {
	get := s.rootGet()
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			get(w)
		default:
			s.handleNotImplemented(w)
		}
	}
}

func (s *Server) handleNotImplemented(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}
