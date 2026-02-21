package fs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	subDir := filepath.Dir(path)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- Scanner Tests ---

func TestScanEmpty(t *testing.T) {
	dir := t.TempDir()
	result, err := Scan(dir, ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(result.Files))
	}
}

func TestScanFindsOfficeFiles(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "report.docx", "word content")
	createTestFile(t, dir, "budget.xlsx", "excel content")
	createTestFile(t, dir, "slides.pptx", "pptx content")
	createTestFile(t, dir, "readme.txt", "not office")

	result, err := Scan(dir, ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 3 {
		t.Errorf("expected 3 office files, got %d", len(result.Files))
	}
	if result.ByFormat["Word"] != 1 {
		t.Errorf("expected 1 Word file")
	}
	if result.ByFormat["Excel"] != 1 {
		t.Errorf("expected 1 Excel file")
	}
	if result.ByFormat["PowerPoint"] != 1 {
		t.Errorf("expected 1 PowerPoint file")
	}
}

func TestScanRecursive(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "top.docx", "top")
	createTestFile(t, dir, "sub/nested.xlsx", "nested")

	// Non-recursive: only top
	result, err := Scan(dir, ScanOptions{Recursive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Errorf("non-recursive: expected 1 file, got %d", len(result.Files))
	}

	// Recursive: both
	result, err = Scan(dir, ScanOptions{Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 2 {
		t.Errorf("recursive: expected 2 files, got %d", len(result.Files))
	}
}

func TestScanFilterExtension(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "report.docx", "word")
	createTestFile(t, dir, "budget.xlsx", "excel")

	result, err := Scan(dir, ScanOptions{Extensions: []string{".docx"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 docx file, got %d", len(result.Files))
	}
	if result.Files[0].Extension != ".docx" {
		t.Errorf("expected .docx, got %q", result.Files[0].Extension)
	}
}

func TestScanWithHash(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "doc.docx", "same content")

	result, err := Scan(dir, ScanOptions{WithHash: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatal("expected 1 file")
	}
	if result.Files[0].SHA256 == "" {
		t.Error("expected SHA256 hash to be set")
	}
}

func TestScanMinMaxSize(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "small.docx", "x")
	createTestFile(t, dir, "big.docx", string(make([]byte, 1024)))

	result, err := Scan(dir, ScanOptions{MinSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file with MinSize=100, got %d", len(result.Files))
	}

	result, err = Scan(dir, ScanOptions{MaxSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file with MaxSize=100, got %d", len(result.Files))
	}
}

func TestScanNotDir(t *testing.T) {
	dir := t.TempDir()
	f := createTestFile(t, dir, "file.txt", "not a dir")
	_, err := Scan(f, ScanOptions{})
	if err == nil {
		t.Fatal("expected error for non-directory")
	}
}

func TestScanTotalSize(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "a.docx", "hello")
	createTestFile(t, dir, "b.xlsx", "world!")

	result, err := Scan(dir, ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalSize != 11 { // 5 + 6
		t.Errorf("TotalSize = %d, want 11", result.TotalSize)
	}
}

// --- Renamer Tests ---

func TestToKebab(t *testing.T) {
	tests := []struct{ in, want string }{
		{"My Report 2025", "my-report-2025"},
		{"camelCase", "camel-case"},
		{"Already-Kebab", "already-kebab"},
		{"  spaces  ", "spaces"},
		{"File (1)", "file-1"},
		{"Q1_Budget_Final", "q1-budget-final"},
	}
	for _, tt := range tests {
		got := toKebab(tt.in)
		if got != tt.want {
			t.Errorf("toKebab(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestToSnake(t *testing.T) {
	got := toSnake("My Report 2025")
	if got != "my_report_2025" {
		t.Errorf("toSnake = %q", got)
	}
}

func TestRenameDryRun(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "My Report.docx", "content")

	files := []FileInfo{
		{
			Path:       filepath.Join(dir, "My Report.docx"),
			Name:       "My Report.docx",
			Extension:  ".docx",
			ModifiedAt: time.Now(),
		},
	}

	results := Rename(files, RenameRule{Pattern: "kebab", DryRun: true})
	if len(results) != 1 {
		t.Fatal("expected 1 result")
	}
	if results[0].Applied {
		t.Error("should not apply in dry run")
	}
	if !containsStr(results[0].NewPath, "my-report.docx") {
		t.Errorf("expected kebab name, got %q", results[0].NewPath)
	}

	// Original should still exist
	if _, err := os.Stat(filepath.Join(dir, "My Report.docx")); err != nil {
		t.Error("original file should still exist in dry run")
	}
}

func TestRenameApply(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "My Report.docx", "content")

	files := []FileInfo{
		{
			Path:       filepath.Join(dir, "My Report.docx"),
			Name:       "My Report.docx",
			Extension:  ".docx",
			ModifiedAt: time.Now(),
		},
	}

	results := Rename(files, RenameRule{Pattern: "kebab", DryRun: false})
	if len(results) != 1 {
		t.Fatal("expected 1 result")
	}
	if !results[0].Applied {
		t.Error("should have applied")
	}

	// New file should exist
	if _, err := os.Stat(filepath.Join(dir, "my-report.docx")); err != nil {
		t.Error("renamed file should exist")
	}
}

func TestRenameDatePrefix(t *testing.T) {
	dir := t.TempDir()
	modTime := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	path := createTestFile(t, dir, "report.docx", "content")
	os.Chtimes(path, modTime, modTime)

	files := []FileInfo{
		{
			Path:       path,
			Name:       "report.docx",
			Extension:  ".docx",
			ModifiedAt: modTime,
		},
	}

	results := Rename(files, RenameRule{Pattern: "date-prefix", DryRun: true})
	if len(results) != 1 {
		t.Fatal("expected 1 result")
	}
	if !containsStr(results[0].NewPath, "2025-03-15-report.docx") {
		t.Errorf("expected date prefix, got %q", results[0].NewPath)
	}
}

func TestRenameNoChange(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "already-kebab.docx", "content")

	files := []FileInfo{
		{
			Path:      filepath.Join(dir, "already-kebab.docx"),
			Name:      "already-kebab.docx",
			Extension: ".docx",
		},
	}

	results := Rename(files, RenameRule{Pattern: "kebab", DryRun: false})
	if results[0].Applied {
		t.Error("should not apply when name doesn't change")
	}
}

// --- Deduper Tests ---

func TestFindDuplicates(t *testing.T) {
	files := []FileInfo{
		{Path: "/a/report.docx", SHA256: "abc123", Size: 1000},
		{Path: "/b/report-copy.docx", SHA256: "abc123", Size: 1000},
		{Path: "/c/unique.xlsx", SHA256: "def456", Size: 2000},
	}

	result := FindDuplicates(files)
	if len(result.Groups) != 1 {
		t.Errorf("expected 1 duplicate group, got %d", len(result.Groups))
	}
	if result.TotalDupes != 1 {
		t.Errorf("expected 1 duplicate, got %d", result.TotalDupes)
	}
	if result.WastedBytes != 1000 {
		t.Errorf("expected 1000 wasted bytes, got %d", result.WastedBytes)
	}
}

func TestFindDuplicatesNone(t *testing.T) {
	files := []FileInfo{
		{Path: "/a/file1.docx", SHA256: "aaa", Size: 100},
		{Path: "/b/file2.docx", SHA256: "bbb", Size: 200},
	}

	result := FindDuplicates(files)
	if len(result.Groups) != 0 {
		t.Errorf("expected no duplicates, got %d groups", len(result.Groups))
	}
}

func TestFindDuplicatesNoHash(t *testing.T) {
	files := []FileInfo{
		{Path: "/a/file1.docx", Size: 100},
		{Path: "/b/file2.docx", Size: 100},
	}

	result := FindDuplicates(files)
	if len(result.Groups) != 0 {
		t.Errorf("expected no duplicates without hashes")
	}
}

func TestRemoveDuplicatesDryRun(t *testing.T) {
	dir := t.TempDir()
	p1 := createTestFile(t, dir, "original.docx", "same")
	p2 := createTestFile(t, dir, "copy.docx", "same")

	groups := []DuplicateGroup{
		{
			SHA256: "abc",
			Size:   4,
			Files: []FileInfo{
				{Path: p1},
				{Path: p2},
			},
		},
	}

	results := RemoveDuplicates(groups, true)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Applied {
		t.Error("should not apply in dry run")
	}

	// Both files should still exist
	if _, err := os.Stat(p2); err != nil {
		t.Error("copy should still exist in dry run")
	}
}

func TestRemoveDuplicatesApply(t *testing.T) {
	dir := t.TempDir()
	p1 := createTestFile(t, dir, "original.docx", "same")
	p2 := createTestFile(t, dir, "copy.docx", "same")

	groups := []DuplicateGroup{
		{
			SHA256: "abc",
			Size:   4,
			Files: []FileInfo{
				{Path: p1},
				{Path: p2},
			},
		},
	}

	results := RemoveDuplicates(groups, false)
	if !results[0].Applied {
		t.Error("should have applied")
	}
	if _, err := os.Stat(p2); !os.IsNotExist(err) {
		t.Error("copy should be deleted")
	}
	if _, err := os.Stat(p1); err != nil {
		t.Error("original should still exist")
	}
}

func TestFormatDedupeReport(t *testing.T) {
	result := &DedupeResult{
		Groups: []DuplicateGroup{
			{SHA256: "abc", Size: 1024, Files: []FileInfo{
				{Path: "/a/file.docx"},
				{Path: "/b/file.docx"},
			}},
		},
		TotalDupes:  1,
		WastedBytes: 1024,
	}

	report := FormatDedupeReport(result)
	if !containsStr(report, "1 duplicate groups") {
		t.Errorf("unexpected report: %s", report)
	}
}

func TestFormatDedupeReportNone(t *testing.T) {
	result := &DedupeResult{}
	report := FormatDedupeReport(result)
	if report != "No duplicates found" {
		t.Errorf("unexpected report: %s", report)
	}
}

// --- Organizer Tests ---

func TestOrganizeByType(t *testing.T) {
	dir := t.TempDir()
	p1 := createTestFile(t, dir, "report.docx", "word")
	p2 := createTestFile(t, dir, "budget.xlsx", "excel")

	files := []FileInfo{
		{Path: p1, Name: "report.docx", Format: "Word"},
		{Path: p2, Name: "budget.xlsx", Format: "Excel"},
	}

	results := OrganizeFile(files, dir, OrganizeRule{Strategy: "by-type", DryRun: true})
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if !containsStr(results[0].NewPath, "Word") {
		t.Errorf("expected Word subdir, got %q", results[0].NewPath)
	}
	if !containsStr(results[1].NewPath, "Excel") {
		t.Errorf("expected Excel subdir, got %q", results[1].NewPath)
	}
}

func TestOrganizeByYear(t *testing.T) {
	dir := t.TempDir()
	p1 := createTestFile(t, dir, "old.docx", "old")

	files := []FileInfo{
		{Path: p1, Name: "old.docx", ModifiedAt: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	results := OrganizeFile(files, dir, OrganizeRule{Strategy: "by-year", DryRun: true})
	if !containsStr(results[0].NewPath, "2023") {
		t.Errorf("expected 2023 subdir, got %q", results[0].NewPath)
	}
}

func TestOrganizeByMonth(t *testing.T) {
	dir := t.TempDir()
	p1 := createTestFile(t, dir, "doc.docx", "content")

	files := []FileInfo{
		{Path: p1, Name: "doc.docx", ModifiedAt: time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)},
	}

	results := OrganizeFile(files, dir, OrganizeRule{Strategy: "by-month", DryRun: true})
	if !containsStr(results[0].NewPath, "2025") || !containsStr(results[0].NewPath, "March") {
		t.Errorf("expected 2025/03-March subdir, got %q", results[0].NewPath)
	}
}

func TestOrganizeApply(t *testing.T) {
	dir := t.TempDir()
	p1 := createTestFile(t, dir, "report.docx", "word")

	files := []FileInfo{
		{Path: p1, Name: "report.docx", Format: "Word"},
	}

	results := OrganizeFile(files, dir, OrganizeRule{Strategy: "by-type", DryRun: false})
	if !results[0].Applied {
		t.Error("should have applied")
	}

	expected := filepath.Join(dir, "Word", "report.docx")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("organized file should exist at %s", expected)
	}
}

func TestStaleFiles(t *testing.T) {
	now := time.Now()
	files := []FileInfo{
		{Path: "/a/old.docx", ModifiedAt: now.Add(-365 * 24 * time.Hour)},
		{Path: "/b/recent.docx", ModifiedAt: now.Add(-1 * time.Hour)},
		{Path: "/c/ancient.docx", ModifiedAt: now.Add(-730 * 24 * time.Hour)},
	}

	stale := StaleFiles(files, 30*24*time.Hour) // 30 days
	if len(stale) != 2 {
		t.Errorf("expected 2 stale files, got %d", len(stale))
	}
	// Should be sorted oldest first
	if stale[0].Path != "/c/ancient.docx" {
		t.Errorf("expected oldest first, got %q", stale[0].Path)
	}
}

func TestManifest(t *testing.T) {
	result := &ScanResult{
		RootDir: "/test",
		Files:   []FileInfo{{Path: "/test/doc.docx", Name: "doc.docx"}},
	}

	data, err := Manifest(result)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "doc.docx") {
		t.Error("manifest should contain file name")
	}
}

func TestFormatSizeFS(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
	}
	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestIsDatePrefixed(t *testing.T) {
	if !isDatePrefixed("2025-01-15-report") {
		t.Error("should detect date prefix")
	}
	if isDatePrefixed("report-2025") {
		t.Error("should not detect non-prefixed")
	}
	if isDatePrefixed("9999-99-99-invalid") {
		t.Error("should reject invalid date")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
