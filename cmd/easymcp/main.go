package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/example/easymcp/internal/config"
	"github.com/example/easymcp/internal/server"
)

const version = "v0.0.1"

func main() {
	log.SetOutput(os.Stdout)

	cfgPath := flag.String("config", "tools.yaml", "path to tool configuration")
	srvName := flag.String("name", "easymcp", "MCP server name")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("EasyMCP " + version)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load tool definitions from YAML
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize and run the server
	srv, err := server.New(cfg, *srvName, version)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
