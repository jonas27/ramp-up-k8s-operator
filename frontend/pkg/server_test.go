package pkg_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jonas27/ramp-up-k8s-operator/frontend/pkg"
	"github.com/matryer/is"
	"go.uber.org/goleak"
	"golang.org/x/exp/slog"
)

func TestRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		path   string
		method string
		gzip   bool
		code   int
	}{
		{"get", "/", http.MethodGet, true, http.StatusOK},
		{"post", "/", http.MethodPost, true, http.StatusOK},
		{"put", "/", http.MethodPut, true, http.StatusNotFound},
		{"no gzip", "/", http.MethodGet, false, http.StatusNotAcceptable},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s := testServer()

			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.gzip {
				req.Header.Set("Accept-Encoding", "gzip")
			}

			w := httptest.NewRecorder()

			serveHTTP(s, w, req)
			is.Equal(w.Code, tt.code)
		})
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func testServer() *pkg.Server {
	const timeout = 1
	// do not log in tests
	log := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	return &pkg.Server{
		Log: log,
		Server: &http.Server{
			Addr:              ":9999",
			ReadHeaderTimeout: timeout * time.Second,
		},
		TemplatePath: "templates",
	}
}

func serveHTTP(s *pkg.Server, w http.ResponseWriter, r *http.Request) {
	s.Routes()
	s.Server.Handler.ServeHTTP(w, r)
}
