// Package fs provides file system intelligence for local Office documents.
package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// OfficeExtensions is the set of recognized Office document extensions.
var OfficeExtensions = map[string]string{
	".docx": "Word",
	".doc":  "Word (Legacy)",
	".xlsx": "Excel",
	".xls":  "Excel (Legacy)",
	".pptx": "PowerPoint",
	".ppt":  "PowerPoint (Legacy)",
	".pdf":  "PDF",
	".odt":  "OpenDocument Text",
	".ods":  "OpenDocument Sheet",
	".odp":  "OpenDocument Presentation",
}

// FileInfo represents a scanned office document.
type FileInfo struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Extension  string    `json:"extension"`
	Format     string    `json:"format"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
	SHA256     string    `json:"sha256,omitempty"`
}

// ScanResult holds the results of a directory scan.
type ScanResult struct {
	RootDir   string            `json:"rootDir"`
	Files     []FileInfo        `json:"files"`
	ByFormat  map[string]int    `json:"byFormat"`
	ByExt     map[string]int    `json:"byExt"`
	TotalSize int64             `json:"totalSize"`
	ScannedAt time.Time         `json:"scannedAt"`
}

// ScanOptions configures the directory scan.
type ScanOptions struct {
	Recursive  bool
	Extensions []string // filter to these extensions; empty = all office
	MinSize    int64
	MaxSize    int64
	ModAfter   time.Time
	ModBefore  time.Time
	WithHash   bool
}

// Scan walks a directory and finds office documents.
func Scan(root string, opts ScanOptions) (*ScanResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("could not access %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	extFilter := make(map[string]bool)
	for _, e := range opts.Extensions {
		e = strings.ToLower(e)
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extFilter[e] = true
	}

	result := &ScanResult{
		RootDir:   root,
		ByFormat:  make(map[string]int),
		ByExt:     make(map[string]int),
		ScannedAt: time.Now(),
	}

	walkFn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if d.IsDir() {
			if !opts.Recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		format, isOffice := OfficeExtensions[ext]
		if !isOffice {
			return nil
		}

		if len(extFilter) > 0 && !extFilter[ext] {
			return nil
		}

		finfo, err := d.Info()
		if err != nil {
			return nil
		}

		if opts.MinSize > 0 && finfo.Size() < opts.MinSize {
			return nil
		}
		if opts.MaxSize > 0 && finfo.Size() > opts.MaxSize {
			return nil
		}
		if !opts.ModAfter.IsZero() && finfo.ModTime().Before(opts.ModAfter) {
			return nil
		}
		if !opts.ModBefore.IsZero() && finfo.ModTime().After(opts.ModBefore) {
			return nil
		}

		fi := FileInfo{
			Path:       path,
			Name:       d.Name(),
			Extension:  ext,
			Format:     format,
			Size:       finfo.Size(),
			ModifiedAt: finfo.ModTime(),
		}

		if opts.WithHash {
			hash, err := hashFile(path)
			if err == nil {
				fi.SHA256 = hash
			}
		}

		result.Files = append(result.Files, fi)
		result.ByFormat[format]++
		result.ByExt[ext]++
		result.TotalSize += finfo.Size()

		return nil
	}

	if err := filepath.WalkDir(root, walkFn); err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	// Sort by path for deterministic output
	sort.Slice(result.Files, func(i, j int) bool {
		return result.Files[i].Path < result.Files[j].Path
	})

	return result, nil
}

// hashFile computes SHA-256 of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
