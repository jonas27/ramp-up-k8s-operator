package pkg //nolint:testpackage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestHTML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		path  string
		debug bool
		panic bool
	}{
		{"correct", "templates", false, false},
		{"correct debug", "templates", true, false},
		{"wrong path", "wrong-path", true, true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Server{
				Debug:        tt.debug,
				Log:          slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
				TemplatePath: tt.path,
			}

			if tt.panic {
				assert.Panics(t, func() { s.html() }, "did not panic on wrong path")
				return
			}

			s.html()
		})
	}
}
