package main

import (
	"context"
	"log"
	"os/exec"

	"github.com/example/easymcp/internal/config"
	"github.com/example/easymcp/internal/executor"
	"github.com/gabriel-vasile/mimetype"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load tool definitions from YAML
	cfg, err := config.Load("tools.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create a server with a single tool.
	server := mcp.NewServer("easymcp", "v0.0.1", nil)
	serverTools := []*mcp.ServerTool{}

	// Register each tool from config
	for _, t := range cfg.Tools {
		name := t.Namespace + "/" + t.Name

		inSchema, err := t.InputSchema()
		if err != nil {
			log.Fatalf("failed to create input schema: %v", err)
		}

		tool := &mcp.ServerTool{
			Tool: &mcp.Tool{
				Name:        name,
				Description: t.Description,
				InputSchema: inSchema,
				// OutputSchema: oschema,
			},
			Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResult, error) {
				var result mcp.CallToolResult
				out, err := executor.RunCommand(ctx, t.Run.Cmd, t.Run.Args, params.Arguments)
				if err != nil {
					switch err.(type) {
					case *exec.ExitError:
						result.IsError = true
					default:
						return nil, err
					}
				}
				switch t.Output.Format {
				case "audio":
					mime := mimetype.Detect(out)
					result.Content = []mcp.Content{&mcp.AudioContent{
						Data:     out,
						MIMEType: mime.String(),
					}}
				case "image":
					mime := mimetype.Detect(out)
					result.Content = []mcp.Content{&mcp.ImageContent{
						Data:     out,
						MIMEType: mime.String(),
					}}
				default:
					result.Content = []mcp.Content{&mcp.TextContent{Text: string(out)}}
				}
				return &result, nil
			},
		}
		serverTools = append(serverTools, tool)
	}

	server.AddTools(serverTools...)

	if err := server.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}
