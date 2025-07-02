package main

import (
	"context"
	"log"

	"github.com/example/easymcp/internal/config"
	"github.com/example/easymcp/internal/executor"
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
	tools := []*mcp.ServerTool{}

	// Register each tool from config
	for _, t := range cfg.Tools {
		name := t.Namespace + "/" + t.Name

		inputOpts := []mcp.SchemaOption{}
		for _, arg := range t.Args {
			argOpts := []mcp.SchemaOption{}
			if arg.Description != "" {
				argOpts = append(argOpts, mcp.Description(arg.Description))
			}
			if len(arg.Enum) > 0 {
				argOpts = append(argOpts, mcp.Enum(arg.Enum...))
			}
			argOpts = append(argOpts, mcp.Required(arg.Required))
			inputOpts = append(inputOpts, mcp.Property(arg.Name, argOpts...))
		}

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
				out, err := executor.RunCommand(ctx, t.Run.Cmd, t.Run.Args, params.Arguments)
				if err != nil {
					return nil, err
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(out)}},
				}, nil
			},
		}

		tools = append(tools, tool)
	}

	server.AddTools(tools...)

	if err := server.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}
