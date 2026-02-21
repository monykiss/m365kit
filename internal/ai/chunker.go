package ai

import (
	"strings"
)

const (
	// DefaultChunkSize is the default maximum number of characters per chunk.
	DefaultChunkSize = 100000
	// DefaultChunkOverlap is the number of overlapping characters between chunks.
	DefaultChunkOverlap = 500
)

// ChunkOptions configures how documents are split into chunks.
type ChunkOptions struct {
	MaxChunkSize int
	Overlap      int
}

// ChunkText splits text into overlapping chunks for processing by AI models
// that have input token limits.
func ChunkText(text string, opts ChunkOptions) []string {
	if opts.MaxChunkSize <= 0 {
		opts.MaxChunkSize = DefaultChunkSize
	}
	if opts.Overlap <= 0 {
		opts.Overlap = DefaultChunkOverlap
	}

	if len(text) <= opts.MaxChunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + opts.MaxChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at a paragraph boundary
		if end < len(text) {
			breakPoint := strings.LastIndex(text[start:end], "\n\n")
			if breakPoint > opts.MaxChunkSize/2 {
				end = start + breakPoint + 2
			} else {
				// Fall back to sentence boundary
				breakPoint = strings.LastIndex(text[start:end], ". ")
				if breakPoint > opts.MaxChunkSize/2 {
					end = start + breakPoint + 2
				}
			}
		}

		chunks = append(chunks, text[start:end])

		// Move start forward, accounting for overlap
		start = end - opts.Overlap
		if start >= len(text) {
			break
		}
	}

	return chunks
}
