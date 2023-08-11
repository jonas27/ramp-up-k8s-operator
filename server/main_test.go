package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	pb "github.com/jonas27/ramp-up-k8s-operator/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener //nolint:gochecknoglobals

func initServer() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterCharacterCounterServer(s, &server{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	dial, err := lis.Dial()
	if err != nil {
		return nil, fmt.Errorf("could not set dialer with error %w", err)
	}
	return dial, nil
}

//nolint:paralleltest
func TestCountCharacters(t *testing.T) {
	initServer()

	t.Setenv("CC_STRING", "testtest1")

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewCharacterCounterClient(conn)
	resp, err := client.CountCharacters(ctx, &pb.CountCharactersRequest{Text: "test this text \n"})
	if err != nil {
		t.Fatalf("CountCharacters failed: %v", err)
	}

	assert.Equal(t, uint64(0x7465737474657374), resp.Characters, "they should be equal")
}
