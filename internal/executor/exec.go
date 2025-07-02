package executor

import (
	"bytes"
	"context"
	"os/exec"
	"text/template"
)

// RunCommand expands a template command and executes it
func RunCommand(ctx context.Context, command string, tmplArgs []string, vars map[string]any) ([]byte, error) {
	args := make([]string, len(tmplArgs))
	for i, arg := range tmplArgs {
		tmpl, err := template.New("cmd").Parse(arg)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars); err != nil {
			return nil, err
		}
		args[i] = buf.String()
	}
	// create subprocess
	cmd := exec.CommandContext(ctx, command, args...)
	return cmd.CombinedOutput()
}
