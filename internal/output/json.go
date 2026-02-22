package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/klytics/m365kit/cmd/version"
)

// Exit codes for consistent error reporting.
const (
	ExitOK          = 0 // success
	ExitUserError   = 1 // bad flags, missing file, auth required
	ExitSystemError = 2 // network failure, IO error, API error
)

// JSONResult is the standard JSON output envelope for all commands.
type JSONResult struct {
	OK      bool        `json:"ok"`
	Command string      `json:"command"`
	Version string      `json:"version"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int         `json:"code,omitempty"`
}

// PrintJSON writes a standard success JSON result to stdout.
func PrintJSON(cmd string, data interface{}) error {
	result := JSONResult{
		OK:      true,
		Command: cmd,
		Version: version.Version,
		Data:    data,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// PrintJSONError writes a standard error JSON result to stdout.
func PrintJSONError(cmd string, err error, code int) error {
	result := JSONResult{
		OK:      false,
		Command: cmd,
		Version: version.Version,
		Error:   err.Error(),
		Code:    code,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(result); encErr != nil {
		return fmt.Errorf("could not encode JSON error: %w", encErr)
	}
	return nil
}
