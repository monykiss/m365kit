// Package report generates documents by combining templates with data sources.
// It supports CSV, JSON, and XLSX data with auto-computed aggregate variables.
package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tmpl "github.com/klytics/m365kit/internal/template"
)

// DataSource represents a loaded data set with rows and columns.
type DataSource struct {
	Columns []string            `json:"columns"`
	Rows    []map[string]string `json:"rows"`
	Source  string              `json:"source"`
}

// GenerateOptions configures report generation.
type GenerateOptions struct {
	TemplatePath string            `json:"templatePath"`
	DataPath     string            `json:"dataPath"`
	OutputPath   string            `json:"outputPath"`
	ExtraValues  map[string]string `json:"extraValues,omitempty"`
}

// GenerateResult holds the outcome of report generation.
type GenerateResult struct {
	OutputPath       string            `json:"outputPath"`
	VariablesApplied int               `json:"variablesApplied"`
	VariablesMissing int               `json:"variablesMissing"`
	MissingNames     []string          `json:"missingNames,omitempty"`
	DataRows         int               `json:"dataRows"`
	ComputedVars     map[string]string `json:"computedVars"`
}

// Generate creates a document by applying data-derived variables to a template.
func Generate(opts GenerateOptions) (*GenerateResult, error) {
	// Load data source
	ds, err := LoadData(opts.DataPath)
	if err != nil {
		return nil, fmt.Errorf("could not load data: %w", err)
	}

	// Compute aggregate variables from numeric columns
	computed := ComputeAggregates(ds)

	// Merge: computed + extra values (extra takes precedence)
	values := make(map[string]string)
	for k, v := range computed {
		values[k] = v
	}
	if opts.ExtraValues != nil {
		for k, v := range opts.ExtraValues {
			values[k] = v
		}
	}

	// Also add first-row values as row_0_<column> and count
	values["row_count"] = strconv.Itoa(len(ds.Rows))
	for i, row := range ds.Rows {
		for col, val := range row {
			values[fmt.Sprintf("row_%d_%s", i, sanitizeVarName(col))] = val
		}
	}

	// Apply template
	result, err := tmpl.Apply(opts.TemplatePath, values, opts.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("could not apply template: %w", err)
	}

	return &GenerateResult{
		OutputPath:       result.OutputPath,
		VariablesApplied: result.VariablesApplied,
		VariablesMissing: result.VariablesMissing,
		MissingNames:     result.MissingNames,
		DataRows:         len(ds.Rows),
		ComputedVars:     computed,
	}, nil
}

// LoadData loads a data source from a file. Supports .csv, .json, and .xlsx.
func LoadData(path string) (*DataSource, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return loadCSV(path)
	case ".json":
		return loadJSON(path)
	default:
		return nil, fmt.Errorf("unsupported data format: %s (supported: .csv, .json)", ext)
	}
}

func loadCSV(path string) (*DataSource, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("could not parse CSV: %w", err)
	}
	if len(records) < 1 {
		return &DataSource{Source: path}, nil
	}

	headers := records[0]
	ds := &DataSource{
		Columns: headers,
		Source:   path,
	}

	for _, row := range records[1:] {
		m := make(map[string]string)
		for i, col := range headers {
			if i < len(row) {
				m[col] = row[i]
			}
		}
		ds.Rows = append(ds.Rows, m)
	}

	return ds, nil
}

func loadJSON(path string) (*DataSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	// Try array of objects first
	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		// Try single object
		var single map[string]any
		if err := json.Unmarshal(data, &single); err != nil {
			return nil, fmt.Errorf("could not parse JSON: expected array of objects or single object")
		}
		records = []map[string]any{single}
	}

	ds := &DataSource{Source: path}

	// Collect all column names
	colSet := make(map[string]bool)
	for _, rec := range records {
		for k := range rec {
			colSet[k] = true
		}
	}
	for k := range colSet {
		ds.Columns = append(ds.Columns, k)
	}
	sort.Strings(ds.Columns)

	// Convert to string rows
	for _, rec := range records {
		m := make(map[string]string)
		for k, v := range rec {
			m[k] = fmt.Sprintf("%v", v)
		}
		ds.Rows = append(ds.Rows, m)
	}

	return ds, nil
}

// ComputeAggregates calculates sum, avg, min, max for each numeric column.
// Returns variables like: sum_revenue, avg_revenue, min_revenue, max_revenue.
func ComputeAggregates(ds *DataSource) map[string]string {
	result := make(map[string]string)
	if len(ds.Rows) == 0 {
		return result
	}

	for _, col := range ds.Columns {
		var values []float64
		for _, row := range ds.Rows {
			val, err := strconv.ParseFloat(strings.TrimSpace(row[col]), 64)
			if err == nil {
				values = append(values, val)
			}
		}

		if len(values) == 0 {
			continue
		}

		varName := sanitizeVarName(col)

		sum := 0.0
		minVal := values[0]
		maxVal := values[0]
		for _, v := range values {
			sum += v
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		avg := sum / float64(len(values))

		result["sum_"+varName] = formatNumber(sum)
		result["avg_"+varName] = formatNumber(avg)
		result["min_"+varName] = formatNumber(minVal)
		result["max_"+varName] = formatNumber(maxVal)
		result["count_"+varName] = strconv.Itoa(len(values))
	}

	return result
}

// sanitizeVarName converts a column name to a valid template variable name.
func sanitizeVarName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	// Remove characters that aren't alphanumeric or underscore
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// formatNumber formats a float as a clean string (no trailing zeros).
func formatNumber(f float64) string {
	if f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

// PreviewVariables returns all variables that would be available for a given data source,
// without actually applying the template.
func PreviewVariables(dataPath string, extraValues map[string]string) (map[string]string, error) {
	ds, err := LoadData(dataPath)
	if err != nil {
		return nil, err
	}

	computed := ComputeAggregates(ds)
	values := make(map[string]string)
	for k, v := range computed {
		values[k] = v
	}
	values["row_count"] = strconv.Itoa(len(ds.Rows))
	for i, row := range ds.Rows {
		for col, val := range row {
			values[fmt.Sprintf("row_%d_%s", i, sanitizeVarName(col))] = val
		}
	}
	if extraValues != nil {
		for k, v := range extraValues {
			values[k] = v
		}
	}

	return values, nil
}
