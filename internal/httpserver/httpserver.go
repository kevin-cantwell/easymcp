package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kevin-cantwell/easymcp/internal/config"
)

// Server exposes the configured tools over HTTP with an OpenAPI spec.
type Server struct {
	cfg     *config.Config
	name    string
	version string
	addr    string
	client  *mcp.ClientSession
	router  *chi.Mux
	spec    *openapi3.T
}

// New creates a new HTTP server for the given tool configuration and MCP client session.
func New(cfg *config.Config, name, version, addr string, client *mcp.ClientSession) (*Server, error) {
	srv := &Server{cfg: cfg, name: name, version: version, addr: addr, client: client}
	r := chi.NewRouter()
	spec := &openapi3.T{
		OpenAPI: "3.1.0",
		Info:    &openapi3.Info{Title: name, Version: version, Description: fmt.Sprintf("%s MCP Server", name)},
		Paths:   openapi3.Paths{},
	}

	for _, t := range cfg.Tools {
		// Build path like /namespace/name
		path := fmt.Sprintf("/%s/%s", t.Namespace, t.Name)
		var reqSchema openapi3.Schema
		if len(t.Input) > 0 {
			inSchema, err := t.InputSchema()
			if err != nil {
				return nil, fmt.Errorf("input schema for %s/%s: %w", t.Namespace, t.Name, err)
			}
			schemaData, err := json.Marshal(inSchema)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(schemaData, &reqSchema); err != nil {
				return nil, err
			}
		}

		// Register HTTP handler
		tt := t
		r.Post(path, func(w http.ResponseWriter, req *http.Request) {
			var args map[string]any
			if len(tt.Input) > 0 {
				if err := json.NewDecoder(req.Body).Decode(&args); err != nil {
					http.Error(w, "invalid json", http.StatusBadRequest)
					return
				}
			}
			res, err := srv.client.CallTool(req.Context(), &mcp.CallToolParams{
				Name:      tt.Namespace + "/" + tt.Name,
				Arguments: args,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(res.Content) == 0 {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// Only handle the first content element
			c := res.Content[0]
			switch v := c.(type) {
			case *mcp.TextContent:
				if tt.Output.Format == "json" {
					w.Header().Set("Content-Type", "application/json")
				} else {
					w.Header().Set("Content-Type", "text/plain")
				}
				w.Write([]byte(v.Text))
			case *mcp.ImageContent:
				if v.MIMEType == "" {
					v.MIMEType = "image/png"
				}
				w.Header().Set("Content-Type", v.MIMEType)
				w.Write(v.Data)
			case *mcp.AudioContent:
				if v.MIMEType == "" {
					v.MIMEType = "audio/mpeg"
				}
				w.Header().Set("Content-Type", v.MIMEType)
				w.Write(v.Data)
			default:
				http.Error(w, "unsupported content type", http.StatusInternalServerError)
			}
		})

		// Add path spec
		okDesc := "OK"
		op := &openapi3.Operation{
			OperationID: tt.Namespace + "_" + tt.Name,
			Summary:     tt.Name,
			Description: tt.Description,
			Responses:   openapi3.Responses{},
		}

		if len(tt.Input) > 0 {
			op.RequestBody = &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{Required: true, Content: openapi3.Content{"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: &reqSchema}}}}}
		}

		respContent := openapi3.Content{}
		switch tt.Output.Format {
		case "json":
			respContent["application/json"] = &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}}
		case "image":
			respContent["image/png"] = &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: openapi3.NewBytesSchema()}}
		case "audio":
			respContent["audio/mpeg"] = &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: openapi3.NewBytesSchema()}}
		default:
			respContent["text/plain"] = &openapi3.MediaType{Schema: &openapi3.SchemaRef{Value: openapi3.NewStringSchema()}}
		}
		op.Responses["200"] = &openapi3.ResponseRef{Value: &openapi3.Response{Description: &okDesc, Content: respContent}}
		if spec.Paths[path] == nil {
			spec.Paths[path] = &openapi3.PathItem{}
		}
		spec.Paths[path].Post = op
	}

	r.Get("/openapi.json", func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(spec)
	})

	srv.router = r
	srv.spec = spec
	return srv, nil
}

// Run starts the HTTP server and blocks until the context is done.
func (s *Server) Run(ctx context.Context) error {
	httpSrv := &http.Server{Addr: s.addr, Handler: s.router}
	go func() {
		<-ctx.Done()
		httpSrv.Shutdown(context.Background())
	}()
	return httpSrv.ListenAndServe()
}
