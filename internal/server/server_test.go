package server

import (
	"context"
	"strings"
	"testing"

	"github.com/kevin-cantwell/easymcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerEcho(t *testing.T) {
	cfg := &config.Config{Tools: []config.Tool{
		{
			Namespace:   "util",
			Name:        "echo",
			Description: "echo",
			Run:         config.Command{Cmd: "echo", Args: []string{"{{.msg}}"}},
			Input:       []config.Input{{Name: "msg", Type: "string", Description: "msg", Required: true}},
			Output:      config.Output{Format: "text"},
		},
	}}

	srv, err := New(cfg, "test", "v0.0.1")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(context.Background(), serverTransport)
	if err != nil {
		t.Fatalf("server connect error: %v", err)
	}
	client := mcp.NewClient("c", "v", nil)
	cs, err := client.Connect(context.Background(), clientTransport)
	if err != nil {
		t.Fatalf("client connect error: %v", err)
	}

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "util/echo",
		Arguments: map[string]any{"msg": "hello"},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	got := strings.TrimSpace(res.Content[0].(*mcp.TextContent).Text)
	if got != "hello" {
		t.Fatalf("unexpected response %q", got)
	}

	cs.Close()
	ss.Wait()
}
