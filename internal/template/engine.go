// Package template provides document template management with variable substitution.
// It handles Word XML run-splitting where {{variable}} may span multiple <w:r> elements.
package template

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Variable represents a template placeholder found in a document.
type Variable struct {
	Name     string `json:"name"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// Template represents a document template with metadata.
type Template struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Path        string     `json:"path"`
	Variables   []Variable `json:"variables"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// ApplyResult holds the outcome of applying variables to a template.
type ApplyResult struct {
	OutputPath       string `json:"outputPath"`
	VariablesApplied int    `json:"variablesApplied"`
	VariablesMissing int    `json:"variablesMissing"`
	MissingNames     []string `json:"missingNames,omitempty"`
}

// Library manages a collection of templates stored on disk.
type Library struct {
	Dir       string     `json:"dir"`
	Templates []Template `json:"templates"`
}

// varPattern matches {{variableName}} with optional whitespace inside braces.
var varPattern = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.]*)\s*\}\}`)

// ExtractVariables scans a .docx file and returns all unique template variables found.
// It handles Word XML run-splitting by merging text across <w:r> elements before scanning.
func ExtractVariables(path string) ([]Variable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}
	return ExtractVariablesFromBytes(data)
}

// ExtractVariablesFromBytes scans raw .docx bytes for template variables.
func ExtractVariablesFromBytes(data []byte) ([]Variable, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid .docx file: %w", err)
	}

	seen := make(map[string]bool)
	var vars []Variable

	for _, f := range reader.File {
		if !isWordXML(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		// Merge runs to handle split variables, then extract
		merged := mergeRunText(string(content))
		matches := varPattern.FindAllStringSubmatch(merged, -1)
		for _, m := range matches {
			name := m[1]
			if !seen[name] {
				seen[name] = true
				vars = append(vars, Variable{Name: name, Required: true})
			}
		}
	}

	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})
	return vars, nil
}

// Apply substitutes template variables in a .docx file and writes the result.
// It handles Word XML run-splitting by consolidating split runs before replacement.
func Apply(templatePath string, values map[string]string, outputPath string) (*ApplyResult, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("could not read template %s: %w", templatePath, err)
	}
	return ApplyFromBytes(data, values, outputPath)
}

// ApplyFromBytes substitutes variables in raw .docx bytes and writes the result.
func ApplyFromBytes(data []byte, values map[string]string, outputPath string) (*ApplyResult, error) {
	result, err := ApplyToBytes(data, values)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("could not create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, result.Data, 0644); err != nil {
		return nil, fmt.Errorf("could not write output %s: %w", outputPath, err)
	}

	return &ApplyResult{
		OutputPath:       outputPath,
		VariablesApplied: result.Applied,
		VariablesMissing: result.Missing,
		MissingNames:     result.MissingNames,
	}, nil
}

// ApplyBytesResult holds the in-memory result of template application.
type ApplyBytesResult struct {
	Data         []byte
	Applied      int
	Missing      int
	MissingNames []string
}

// ApplyToBytes substitutes variables in raw .docx bytes and returns the result in memory.
func ApplyToBytes(data []byte, values map[string]string) (*ApplyBytesResult, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid .docx file: %w", err)
	}

	// First pass: find all variable names used
	allVars := make(map[string]bool)
	for _, f := range reader.File {
		if !isWordXML(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		merged := mergeRunText(string(content))
		for _, m := range varPattern.FindAllStringSubmatch(merged, -1) {
			allVars[m[1]] = true
		}
	}

	// Calculate missing
	var missingNames []string
	for name := range allVars {
		if _, ok := values[name]; !ok {
			missingNames = append(missingNames, name)
		}
	}
	sort.Strings(missingNames)

	// Re-read and apply
	reader, _ = zip.NewReader(bytes.NewReader(data), int64(len(data)))
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	applied := 0

	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("could not open %s: %w", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("could not read %s: %w", f.Name, err)
		}

		if isWordXML(f.Name) {
			text := string(content)
			// Fix run-splitting: consolidate fragmented {{variable}} patterns
			text = fixRunSplitting(text)
			// Now perform substitutions on the consolidated text
			for name, value := range values {
				placeholder := "{{" + name + "}}"
				count := strings.Count(text, placeholder)
				if count > 0 {
					applied += count
					text = strings.ReplaceAll(text, placeholder, xmlEscape(value))
				}
			}
			content = []byte(text)
		}

		header := &zip.FileHeader{
			Name:     f.Name,
			Method:   f.Method,
			Modified: f.Modified,
		}
		w, err := writer.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("could not create %s: %w", f.Name, err)
		}
		if _, err := w.Write(content); err != nil {
			return nil, fmt.Errorf("could not write %s: %w", f.Name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("could not finalize output: %w", err)
	}

	return &ApplyBytesResult{
		Data:         buf.Bytes(),
		Applied:      applied,
		Missing:      len(missingNames),
		MissingNames: missingNames,
	}, nil
}

// fixRunSplitting handles the Word XML run-splitting problem.
// Word often splits {{variable}} across multiple <w:r> elements like:
//
//	<w:r><w:t>{{</w:t></w:r><w:r><w:t>name</w:t></w:r><w:r><w:t>}}</w:t></w:r>
//
// This function consolidates such split runs into a single run containing the complete
// variable reference, preserving surrounding XML structure.
func fixRunSplitting(xmlText string) string {
	// Strategy: find sequences of <w:r>...</w:r> elements within the same paragraph
	// where the concatenated text forms a {{variable}} pattern, and merge them.

	// Match individual runs: <w:r>...<w:t ...>TEXT</w:t>...</w:r>
	runPattern := regexp.MustCompile(`<w:r\b[^>]*>(?:<w:rPr>.*?</w:rPr>)?<w:t[^>]*>([^<]*)</w:t></w:r>`)

	// Process paragraph by paragraph
	paraPattern := regexp.MustCompile(`(?s)(<w:p\b[^>]*>)(.*?)(</w:p>)`)

	return paraPattern.ReplaceAllStringFunc(xmlText, func(para string) string {
		submatches := paraPattern.FindStringSubmatch(para)
		if submatches == nil {
			return para
		}
		paraOpen := submatches[1]
		paraBody := submatches[2]
		paraClose := submatches[3]

		// Find all runs in this paragraph
		runMatches := runPattern.FindAllStringSubmatchIndex(paraBody, -1)
		if len(runMatches) < 2 {
			return para
		}

		// Extract run positions and their text content
		type runInfo struct {
			fullStart, fullEnd int
			text               string
		}
		var runs []runInfo
		for _, loc := range runMatches {
			runs = append(runs, runInfo{
				fullStart: loc[0],
				fullEnd:   loc[1],
				text:      paraBody[loc[2]:loc[3]],
			})
		}

		// Look for sequences of consecutive runs whose concatenated text
		// contains a {{variable}} pattern
		result := paraBody
		offset := 0
		merged := false

		for i := 0; i < len(runs); i++ {
			// Check if this run starts or contains part of a {{ pattern
			if !strings.Contains(runs[i].text, "{") && !strings.Contains(runs[i].text, "}") {
				continue
			}

			// Try concatenating from this run forward
			for j := i + 1; j <= len(runs) && j <= i+10; j++ {
				var combined strings.Builder
				for k := i; k < j; k++ {
					combined.WriteString(runs[k].text)
				}
				combinedText := combined.String()

				if varPattern.MatchString(combinedText) && j > i+1 {
					// Found a split variable! Merge runs i through j-1
					// Replace the entire sequence with a single run containing the merged text
					firstRunStart := runs[i].fullStart + offset
					lastRunEnd := runs[j-1].fullEnd + offset

					// Build the replacement: use the first run's structure but with merged text
					replacement := `<w:r><w:t xml:space="preserve">` + combinedText + `</w:t></w:r>`
					original := result[firstRunStart:lastRunEnd]

					result = result[:firstRunStart] + replacement + result[lastRunEnd:]
					offset += len(replacement) - len(original)
					merged = true

					// Skip the runs we just merged
					i = j - 1
					break
				}

				// If we've already found the closing }}, no point continuing
				if strings.Contains(combinedText, "}}") {
					break
				}
			}
		}

		if merged {
			return paraOpen + result + paraClose
		}
		return para
	})
}

// mergeRunText extracts and concatenates all text from <w:t> elements within runs,
// used for variable detection (not for output).
func mergeRunText(xmlText string) string {
	// For extraction purposes, just concatenate all <w:t> text content
	textPattern := regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)
	matches := textPattern.FindAllStringSubmatch(xmlText, -1)
	var b strings.Builder
	for _, m := range matches {
		b.WriteString(m[1])
	}
	return b.String()
}

func isWordXML(name string) bool {
	return strings.HasPrefix(name, "word/") && strings.HasSuffix(name, ".xml")
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// Library functions

const libraryFile = "templates.json"

// LoadLibrary loads the template library from the given directory.
func LoadLibrary(dir string) (*Library, error) {
	lib := &Library{Dir: dir}
	path := filepath.Join(dir, libraryFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return lib, nil
		}
		return nil, fmt.Errorf("could not read library: %w", err)
	}

	if err := json.Unmarshal(data, &lib.Templates); err != nil {
		return nil, fmt.Errorf("could not parse library: %w", err)
	}
	return lib, nil
}

// Save persists the library to disk.
func (lib *Library) Save() error {
	if err := os.MkdirAll(lib.Dir, 0755); err != nil {
		return fmt.Errorf("could not create library directory: %w", err)
	}

	data, err := json.MarshalIndent(lib.Templates, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal library: %w", err)
	}

	return os.WriteFile(filepath.Join(lib.Dir, libraryFile), data, 0644)
}

// Add registers a new template in the library.
func (lib *Library) Add(name, description, docxPath string) (*Template, error) {
	// Check for duplicates
	for _, t := range lib.Templates {
		if t.Name == name {
			return nil, fmt.Errorf("template %q already exists", name)
		}
	}

	// Validate file exists
	absPath, err := filepath.Abs(docxPath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("file not found: %s", absPath)
	}

	// Extract variables
	vars, err := ExtractVariables(absPath)
	if err != nil {
		return nil, fmt.Errorf("could not extract variables: %w", err)
	}

	now := time.Now()
	tmpl := Template{
		Name:        name,
		Description: description,
		Path:        absPath,
		Variables:   vars,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	lib.Templates = append(lib.Templates, tmpl)
	if err := lib.Save(); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// Remove deletes a template from the library by name.
func (lib *Library) Remove(name string) error {
	for i, t := range lib.Templates {
		if t.Name == name {
			lib.Templates = append(lib.Templates[:i], lib.Templates[i+1:]...)
			return lib.Save()
		}
	}
	return fmt.Errorf("template %q not found", name)
}

// Get returns a template by name.
func (lib *Library) Get(name string) (*Template, error) {
	for _, t := range lib.Templates {
		if t.Name == name {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("template %q not found", name)
}

// List returns all templates sorted by name.
func (lib *Library) List() []Template {
	sorted := make([]Template, len(lib.Templates))
	copy(sorted, lib.Templates)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

// DefaultLibraryDir returns the default template library directory.
func DefaultLibraryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kit", "templates")
}
