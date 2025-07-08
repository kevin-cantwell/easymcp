package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/example/easymcp/internal/config"
	"github.com/example/easymcp/internal/httpserver"
	"github.com/example/easymcp/internal/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "v0.0.1"

func main() {
	log.SetOutput(os.Stdout)

	cfgPath := flag.String("config", "tools.yaml", "path to tool configuration")
	srvName := flag.String("name", "easymcp", "MCP server name")
	port := flag.Int("port", 0, "start HTTP server on this port instead of stdio")
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

	srv, err := server.New(cfg, *srvName, version)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if *port != 0 {
		if *port < 1000 || *port > 9999 {
			log.Fatal("port must be a 4 digit number")
		}
		addr := fmt.Sprintf(":%04d", *port)

		// create in-memory connection for HTTP proxy
		cTransport, sTransport := mcp.NewInMemoryTransports()
		ss, err := srv.Connect(ctx, sTransport)
		if err != nil {
			log.Fatalf("failed to connect in-memory: %v", err)
		}
		client := mcp.NewClient(*srvName, version, nil)
		cs, err := client.Connect(ctx, cTransport)
		if err != nil {
			log.Fatalf("failed to create client: %v", err)
		}

		// start stdio server
		go func() {
			if err := srv.Run(ctx); err != nil && err != context.Canceled {
				log.Println(err)
			}
		}()

		httpSrv, err := httpserver.New(cfg, *srvName, version, addr, cs)
		if err != nil {
			log.Fatalf("failed to init http server: %v", err)
		}
		go func() {
			<-ctx.Done()
			cs.Close()
		}()
		if err := httpSrv.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		cs.Close()
		ss.Wait()
		return
	}

	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
