package main

import (
	"context"
	"log"
	"sync"

	"github.com/example/easymcp/internal/config"
	"github.com/example/easymcp/internal/executor"
	"github.com/fsnotify/fsnotify"
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

	server := mcp.NewServer("easymcp", "v0.0.1", nil)

	tools, err := buildServerTools(cfg)
	if err != nil {
		log.Fatalf("failed to create tools: %v", err)
	}
	server.AddTools(tools...)

	if err := startConfigWatcher(ctx, "tools.yaml", server); err != nil {
		log.Fatalf("failed to watch config: %v", err)
	}

	if err := server.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}

func buildServerTools(cfg *config.Config) ([]*mcp.ServerTool, error) {
	tools := make([]*mcp.ServerTool, 0, len(cfg.Tools))
	for _, t := range cfg.Tools {
		t := t
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
			return nil, err
		}

		tool := &mcp.ServerTool{
			Tool: &mcp.Tool{
				Name:        name,
				Description: t.Description,
				InputSchema: inSchema,
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
	return tools, nil
}

func startConfigWatcher(ctx context.Context, path string, server *mcp.Server) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(path); err != nil {
		return err
	}

	var mu sync.Mutex
	var current *config.Config
	current, _ = config.Load(path)

	go func() {
		defer watcher.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					cfg, err := config.Load(path)
					if err != nil {
						log.Printf("failed to reload config: %v", err)
						continue
					}
					mu.Lock()
					if current != nil {
						names := make([]string, len(current.Tools))
						for i, t := range current.Tools {
							names[i] = t.Namespace + "/" + t.Name
						}
						server.RemoveTools(names...)
					}
					tools, err := buildServerTools(cfg)
					if err != nil {
						mu.Unlock()
						log.Printf("failed to create tools: %v", err)
						continue
					}
					server.AddTools(tools...)
					current = cfg
					mu.Unlock()
					log.Printf("reloaded tools from %s", path)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)
			}
		}
	}()
	return nil
}
