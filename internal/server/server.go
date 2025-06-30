package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// Tool represents a single MCP tool.
type Tool struct {
	Name        string
	Description string
	// ArgsSchema describes the JSON schema for tool parameters.
	ArgsSchema *openapi3.SchemaRef
}

// GenerateOpenAPISpec returns an OpenAPI 3.1 document describing the tools.
func GenerateOpenAPISpec(tools []Tool) (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI: "3.1.0",
		Info: &openapi3.Info{
			Title:       "EasyMCP",
			Version:     "0.1.0",
			Description: "OpenAPI description for EasyMCP tools",
		},
		Paths:      openapi3.NewPaths(),
		Components: &openapi3.Components{Schemas: openapi3.Schemas{}},
	}

	for _, tool := range tools {
		// Each tool is represented as an RPC method under /rpc
		op := &openapi3.Operation{
			OperationID: tool.Name,
			Summary:     tool.Description,
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(http.StatusOK, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("Success")}),
			),
		}
		if tool.ArgsSchema != nil {
			doc.Components.Schemas[tool.Name+"Args"] = tool.ArgsSchema
			op.RequestBody = &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
				Required: true,
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Ref: "#/components/schemas/" + tool.Name + "Args"},
					},
				},
			}}
		}
		if item := doc.Paths.Value("/rpc"); item != nil {
			item.Post = op
		} else {
			doc.Paths.Set("/rpc", &openapi3.PathItem{Post: op})
		}
	}

	return doc, nil
}

func ptr[T any](v T) *T { return &v }

// Run serves the MCP JSON-RPC handler and OpenAPI document on the given address.
func Run(ctx context.Context, addr string, tools []Tool, rpcHandler http.Handler) error {
	mux := http.NewServeMux()

	// RPC endpoint
	mux.Handle("/rpc", rpcHandler)

	// OpenAPI endpoint
	openAPIDoc, err := GenerateOpenAPISpec(tools)
	if err != nil {
		return err
	}
	openAPIDocJSON, err := json.Marshal(openAPIDoc)
	if err != nil {
		return err
	}
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(openAPIDocJSON)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		ctxShutdown, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := srv.Shutdown(ctxShutdown); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	return srv.ListenAndServe()
}
