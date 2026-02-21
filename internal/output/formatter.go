// Package output provides formatting utilities for CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Format represents an output format.
type Format int

const (
	// FormatText is plain text output.
	FormatText Format = iota
	// FormatJSON is JSON output.
	FormatJSON
	// FormatMarkdown is Markdown output.
	FormatMarkdown
)

// Writer handles formatted output to a destination.
type Writer struct {
	dest   io.Writer
	format Format
}

// NewWriter creates a new output writer with the given format.
func NewWriter(format Format) *Writer {
	return &Writer{
		dest:   os.Stdout,
		format: format,
	}
}

// WriteJSON encodes a value as pretty-printed JSON.
func (w *Writer) WriteJSON(v interface{}) error {
	enc := json.NewEncoder(w.dest)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteText writes plain text.
func (w *Writer) WriteText(s string) error {
	_, err := fmt.Fprint(w.dest, s)
	return err
}

// WriteLn writes a line of text.
func (w *Writer) WriteLn(s string) error {
	_, err := fmt.Fprintln(w.dest, s)
	return err
}

// WriteError writes an error message to stderr.
func WriteError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
