package google

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/deenaik/gtd-cli/internal/config"
)

// Run executes a gog command and returns raw output.
func Run(args ...string) ([]byte, error) {
	cfg := config.Get()
	cmd := exec.Command(cfg.GogPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gog %v: %s: %w", args, stderr.String(), err)
	}
	return stdout.Bytes(), nil
}

// RunJSON executes gog with the -j flag and unmarshals the JSON output into result.
func RunJSON(result any, args ...string) error {
	args = append([]string{"-j", "--results-only"}, args...)
	out, err := Run(args...)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, result)
}
