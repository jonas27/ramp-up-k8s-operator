package pkg

import (
	"context"
	"net/http"
	"strings"
	"time"

	pb "github.com/jonas27/ramp-up-k8s-operator/proto"

	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Server holds configurable parameters.
type Server struct {
	Debug        bool
	Log          *slog.Logger
	Server       *http.Server
	GRPCAddr     string
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
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			http.Error(w, "enable gzip to use this", http.StatusNotAcceptable)
			return
		}
		switch r.Method {
		case http.MethodGet:
			html(w, 0)
		case http.MethodPost:
			defer r.Body.Close()
			if err := r.ParseMultipartForm(0); err != nil {
				s.Log.Error("Error while parsing post multipart form", "error", err)
			}

			text := r.FormValue("text")

			s.Log.Info("dialing grpc server", "addr", s.GRPCAddr)
			conn, err := grpc.Dial(s.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				s.Log.Error("failed to dial grpc server", "error", err)
				return
			}
			defer conn.Close()

			s.Log.Info("test", "addr", s.GRPCAddr)

			client := pb.NewCharacterCounterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			s.Log.Info("test1", "addr", s.GRPCAddr)
			resp, err := client.CountCharacters(ctx, &pb.CountCharactersRequest{Text: text})
			if err != nil {
				s.Log.Error("errored during CountCharacters request", "error", err)
				return
			}

			html(w, resp.Characters)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
