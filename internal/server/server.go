package server

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/gabriel-vasile/mimetype"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kevin-cantwell/easymcp/internal/config"
	"github.com/kevin-cantwell/easymcp/internal/executor"
)

type Server struct {
	*mcp.Server
}

func New(cfg *config.Config, name, version string) (*Server, error) {
	srv := mcp.NewServer(name, version, nil)
	var serverTools []*mcp.ServerTool
	for _, t := range cfg.Tools {
		name := t.Namespace + "/" + t.Name
		inSchema, err := t.InputSchema()
		if err != nil {
			return nil, fmt.Errorf("failed to create input schema: %w", err)
		}
		tool := &mcp.ServerTool{
			Tool: &mcp.Tool{
				Name:        name,
				Description: t.Description,
				InputSchema: inSchema,
			},
		}
		// attach handler separately to capture t
		tool.Handler = func(t config.Tool) mcp.ToolHandler {
			return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResult, error) {
				var result mcp.CallToolResult
				out, err := executor.RunCommand(ctx, t.Run.Cmd, t.Run.Args, params.Arguments)
				if err != nil {
					log.Printf("Error running command %s %v %v: %v\n", t.Run.Cmd, t.Run.Args, params.Arguments, err)
					result.IsError = true
					switch err.(type) {
					case *exec.ExitError:
						result.Content = []mcp.Content{&mcp.TextContent{Text: string(out)}}
					default:
						result.Content = []mcp.Content{&mcp.TextContent{
							Text: fmt.Sprintf("tool error: failed to run command: %s", t.Run.Cmd),
						}}
					}
					return &result, nil
				}
				switch t.Output.Format {
				case "audio":
					mime := mimetype.Detect(out)
					result.Content = []mcp.Content{&mcp.AudioContent{Data: out, MIMEType: mime.String()}}
				case "image":
					mime := mimetype.Detect(out)
					result.Content = []mcp.Content{&mcp.ImageContent{Data: out, MIMEType: mime.String()}}
				default:
					result.Content = []mcp.Content{&mcp.TextContent{Text: string(out)}}
				}
				return &result, nil
			}
		}(t)
		serverTools = append(serverTools, tool)
	}
	srv.AddTools(serverTools...)
	return &Server{srv}, nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.Server.Run(ctx, mcp.NewStdioTransport())
}
