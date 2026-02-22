package template

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeDocx creates a minimal .docx with the given document.xml body content.
func makeDocx(bodyContent string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// [Content_Types].xml
	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(xml.Header + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	// _rels/.rels
	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	// word/document.xml
	w, _ = zw.Create("word/document.xml")
	w.Write([]byte(xml.Header + `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		bodyContent +
		`</w:body></w:document>`))

	zw.Close()
	return buf.Bytes()
}

func TestExtractVariablesSimple(t *testing.T) {
	body := `<w:p><w:r><w:t>Hello {{name}}, welcome to {{company}}!</w:t></w:r></w:p>`
	data := makeDocx(body)

	vars, err := ExtractVariablesFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(vars))
	}
	// Sorted alphabetically
	if vars[0].Name != "company" {
		t.Errorf("expected company, got %q", vars[0].Name)
	}
	if vars[1].Name != "name" {
		t.Errorf("expected name, got %q", vars[1].Name)
	}
}

func TestExtractVariablesNone(t *testing.T) {
	body := `<w:p><w:r><w:t>No variables here.</w:t></w:r></w:p>`
	data := makeDocx(body)

	vars, err := ExtractVariablesFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 0 {
		t.Errorf("expected 0 variables, got %d", len(vars))
	}
}

func TestExtractVariablesDeduplicate(t *testing.T) {
	body := `<w:p><w:r><w:t>{{name}} and {{name}} and {{name}}</w:t></w:r></w:p>`
	data := makeDocx(body)

	vars, err := ExtractVariablesFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 1 {
		t.Fatalf("expected 1 unique variable, got %d", len(vars))
	}
	if vars[0].Name != "name" {
		t.Errorf("expected name, got %q", vars[0].Name)
	}
}

func TestExtractVariablesWithSpaces(t *testing.T) {
	body := `<w:p><w:r><w:t>{{ name }} and {{  company  }}</w:t></w:r></w:p>`
	data := makeDocx(body)

	vars, err := ExtractVariablesFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(vars))
	}
}

func TestApplySimple(t *testing.T) {
	body := `<w:p><w:r><w:t>Dear {{name}}, your order {{order_id}} is ready.</w:t></w:r></w:p>`
	data := makeDocx(body)

	values := map[string]string{
		"name":     "Alice",
		"order_id": "ORD-12345",
	}

	result, err := ApplyToBytes(data, values)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 2 {
		t.Errorf("expected 2 applied, got %d", result.Applied)
	}
	if result.Missing != 0 {
		t.Errorf("expected 0 missing, got %d", result.Missing)
	}

	// Verify the output is a valid docx with replaced content
	reader, err := zip.NewReader(bytes.NewReader(result.Data), int64(len(result.Data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			text := string(content)
			if !strings.Contains(text, "Alice") {
				t.Error("expected replaced name 'Alice' in output")
			}
			if !strings.Contains(text, "ORD-12345") {
				t.Error("expected replaced order_id in output")
			}
			if strings.Contains(text, "{{name}}") {
				t.Error("variable {{name}} should have been replaced")
			}
		}
	}
}

func TestApplyMissingVariables(t *testing.T) {
	body := `<w:p><w:r><w:t>Hello {{name}}, welcome to {{company}}!</w:t></w:r></w:p>`
	data := makeDocx(body)

	values := map[string]string{
		"name": "Alice",
		// company is missing
	}

	result, err := ApplyToBytes(data, values)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Errorf("expected 1 applied, got %d", result.Applied)
	}
	if result.Missing != 1 {
		t.Errorf("expected 1 missing, got %d", result.Missing)
	}
	if len(result.MissingNames) != 1 || result.MissingNames[0] != "company" {
		t.Errorf("expected missing=[company], got %v", result.MissingNames)
	}
}

func TestApplyXMLEscape(t *testing.T) {
	body := `<w:p><w:r><w:t>Company: {{company}}</w:t></w:r></w:p>`
	data := makeDocx(body)

	values := map[string]string{
		"company": "Smith & Jones <Legal>",
	}

	result, err := ApplyToBytes(data, values)
	if err != nil {
		t.Fatal(err)
	}

	reader, _ := zip.NewReader(bytes.NewReader(result.Data), int64(len(result.Data)))
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			text := string(content)
			if !strings.Contains(text, "Smith &amp; Jones &lt;Legal&gt;") {
				t.Errorf("expected XML-escaped value, got: %s", text)
			}
		}
	}
}

func TestFixRunSplitting(t *testing.T) {
	// Simulate Word splitting {{name}} across 3 runs
	body := `<w:p>` +
		`<w:r><w:t>Hello </w:t></w:r>` +
		`<w:r><w:t>{{</w:t></w:r>` +
		`<w:r><w:t>name</w:t></w:r>` +
		`<w:r><w:t>}}</w:t></w:r>` +
		`<w:r><w:t>, welcome!</w:t></w:r>` +
		`</w:p>`
	data := makeDocx(body)

	values := map[string]string{"name": "Bob"}

	result, err := ApplyToBytes(data, values)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Errorf("expected 1 applied after run-splitting fix, got %d", result.Applied)
	}

	reader, _ := zip.NewReader(bytes.NewReader(result.Data), int64(len(result.Data)))
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			text := string(content)
			if !strings.Contains(text, "Bob") {
				t.Error("expected 'Bob' in output after run-splitting fix")
			}
		}
	}
}

func TestFixRunSplittingTwoRuns(t *testing.T) {
	// Word splits {{name}} across 2 runs: "{{na" and "me}}"
	body := `<w:p>` +
		`<w:r><w:t>Dear </w:t></w:r>` +
		`<w:r><w:t>{{na</w:t></w:r>` +
		`<w:r><w:t>me}}</w:t></w:r>` +
		`<w:r><w:t> regards</w:t></w:r>` +
		`</w:p>`
	data := makeDocx(body)

	values := map[string]string{"name": "Charlie"}

	result, err := ApplyToBytes(data, values)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Errorf("expected 1 applied, got %d", result.Applied)
	}
}

func TestApplyMultipleOccurrences(t *testing.T) {
	body := `<w:p><w:r><w:t>{{name}} spoke to {{name}} about {{topic}}</w:t></w:r></w:p>`
	data := makeDocx(body)

	values := map[string]string{
		"name":  "Alice",
		"topic": "budget",
	}

	result, err := ApplyToBytes(data, values)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 3 {
		t.Errorf("expected 3 applied (2x name + 1x topic), got %d", result.Applied)
	}
}

func TestApplyToFile(t *testing.T) {
	body := `<w:p><w:r><w:t>Hello {{name}}</w:t></w:r></w:p>`
	data := makeDocx(body)

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.docx")
	outputPath := filepath.Join(dir, "output.docx")

	os.WriteFile(templatePath, data, 0644)

	result, err := Apply(templatePath, map[string]string{"name": "Dana"}, outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputPath != outputPath {
		t.Errorf("expected output path %q, got %q", outputPath, result.OutputPath)
	}
	if result.VariablesApplied != 1 {
		t.Errorf("expected 1 applied, got %d", result.VariablesApplied)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

func TestLibraryAddAndList(t *testing.T) {
	dir := t.TempDir()

	// Create a template docx
	body := `<w:p><w:r><w:t>Hello {{name}} at {{company}}</w:t></w:r></w:p>`
	data := makeDocx(body)
	templatePath := filepath.Join(dir, "greeting.docx")
	os.WriteFile(templatePath, data, 0644)

	lib, err := LoadLibrary(dir)
	if err != nil {
		t.Fatal(err)
	}

	tmpl, err := lib.Add("greeting", "A greeting template", templatePath)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Name != "greeting" {
		t.Errorf("expected name 'greeting', got %q", tmpl.Name)
	}
	if len(tmpl.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(tmpl.Variables))
	}

	// List
	templates := lib.List()
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}

	// Get
	got, err := lib.Get("greeting")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "A greeting template" {
		t.Errorf("unexpected description: %q", got.Description)
	}
}

func TestLibraryRemove(t *testing.T) {
	dir := t.TempDir()

	body := `<w:p><w:r><w:t>{{x}}</w:t></w:r></w:p>`
	data := makeDocx(body)
	templatePath := filepath.Join(dir, "t.docx")
	os.WriteFile(templatePath, data, 0644)

	lib, _ := LoadLibrary(dir)
	lib.Add("test", "desc", templatePath)

	if err := lib.Remove("test"); err != nil {
		t.Fatal(err)
	}
	if len(lib.Templates) != 0 {
		t.Errorf("expected 0 templates after remove, got %d", len(lib.Templates))
	}
}

func TestLibraryDuplicate(t *testing.T) {
	dir := t.TempDir()

	body := `<w:p><w:r><w:t>{{x}}</w:t></w:r></w:p>`
	data := makeDocx(body)
	templatePath := filepath.Join(dir, "t.docx")
	os.WriteFile(templatePath, data, 0644)

	lib, _ := LoadLibrary(dir)
	lib.Add("test", "desc", templatePath)

	_, err := lib.Add("test", "desc2", templatePath)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestLibraryPersistence(t *testing.T) {
	dir := t.TempDir()

	body := `<w:p><w:r><w:t>{{greeting}}</w:t></w:r></w:p>`
	data := makeDocx(body)
	templatePath := filepath.Join(dir, "t.docx")
	os.WriteFile(templatePath, data, 0644)

	lib1, _ := LoadLibrary(dir)
	lib1.Add("hello", "Say hello", templatePath)

	// Reload from disk
	lib2, err := LoadLibrary(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(lib2.Templates) != 1 {
		t.Fatalf("expected 1 template after reload, got %d", len(lib2.Templates))
	}
	if lib2.Templates[0].Name != "hello" {
		t.Errorf("expected 'hello', got %q", lib2.Templates[0].Name)
	}
}

func TestVarPattern(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"{{name}}", []string{"name"}},
		{"{{ name }}", []string{"name"}},
		{"{{first_name}}", []string{"first_name"}},
		{"{{company.name}}", []string{"company.name"}},
		{"{{a}} and {{b}}", []string{"a", "b"}},
		{"no vars here", nil},
		{"{single}", nil},
		{"{{123invalid}}", nil}, // starts with number
	}

	for _, tt := range tests {
		matches := varPattern.FindAllStringSubmatch(tt.input, -1)
		var got []string
		for _, m := range matches {
			got = append(got, m[1])
		}
		if len(got) != len(tt.want) {
			t.Errorf("varPattern(%q): got %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("varPattern(%q)[%d]: got %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMergeRunText(t *testing.T) {
	input := `<w:r><w:t>Hello </w:t></w:r><w:r><w:t>World</w:t></w:r>`
	got := mergeRunText(input)
	if got != "Hello World" {
		t.Errorf("mergeRunText: got %q, want %q", got, "Hello World")
	}
}

func TestTemplateJSON(t *testing.T) {
	tmpl := Template{
		Name:        "invoice",
		Description: "Invoice template",
		Variables:   []Variable{{Name: "amount", Required: true}},
	}

	data, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Template
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "invoice" {
		t.Errorf("expected 'invoice', got %q", decoded.Name)
	}
	if len(decoded.Variables) != 1 {
		t.Errorf("expected 1 variable, got %d", len(decoded.Variables))
	}
}

