package executor

import (
	"context"
	"strings"
	"testing"
)

func TestRunCommand(t *testing.T) {
	ctx := context.Background()
	out, err := RunCommand(ctx, "echo", []string{"{{.msg}}"}, map[string]any{"msg": "hello"})
	if err != nil {
		t.Fatalf("RunCommand returned error: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "hello" {
		t.Fatalf("expected output 'hello', got %q", got)
	}
}
