package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	data := `tools:
  - namespace: util
    name: echo
    description: echo message
    run:
      cmd: echo
      args: ["{{.msg}}"]
    input:
      - name: msg
        description: message
        type: string
        required: true
    output:
      format: text
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "echo" {
		t.Fatalf("unexpected tool name %q", cfg.Tools[0].Name)
	}
}

func TestToolInputSchema(t *testing.T) {
	tool := Tool{
		Namespace:   "util",
		Name:        "echo",
		Description: "echo",
		Run:         Command{Cmd: "echo", Args: []string{"{{.msg}}"}},
		Input:       []Input{{Name: "msg", Type: "string", Description: "msg", Required: true}},
		Output:      Output{Format: "text"},
	}
	schema, err := tool.InputSchema()
	if err != nil {
		t.Fatalf("InputSchema returned error: %v", err)
	}
	if schema.Properties["msg"] == nil {
		t.Fatalf("expected property 'msg' in schema")
	}
	found := false
	for _, req := range schema.Required {
		if req == "msg" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'msg' to be required")
	}
}
