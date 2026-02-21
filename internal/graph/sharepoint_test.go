package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewSharePoint(t *testing.T) {
	client := &http.Client{}
	sp := NewSharePoint(client)
	if sp == nil {
		t.Fatal("expected non-nil SharePoint")
	}
	if sp.Client != client {
		t.Error("client mismatch")
	}
}

func TestSiteJSON(t *testing.T) {
	raw := `{
		"id": "site-123",
		"displayName": "Marketing",
		"name": "marketing",
		"webUrl": "https://example.sharepoint.com/sites/marketing",
		"description": "Marketing team site"
	}`

	var site Site
	if err := json.Unmarshal([]byte(raw), &site); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if site.ID != "site-123" {
		t.Errorf("ID = %q", site.ID)
	}
	if site.DisplayName != "Marketing" {
		t.Errorf("DisplayName = %q", site.DisplayName)
	}
	if site.WebURL != "https://example.sharepoint.com/sites/marketing" {
		t.Errorf("WebURL = %q", site.WebURL)
	}
}

func TestDocumentLibraryJSON(t *testing.T) {
	raw := `{
		"id": "drive-456",
		"displayName": "Documents",
		"name": "Shared Documents",
		"webUrl": "https://example.sharepoint.com/sites/marketing/Shared%20Documents",
		"driveType": "documentLibrary"
	}`

	var lib DocumentLibrary
	if err := json.Unmarshal([]byte(raw), &lib); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if lib.ID != "drive-456" {
		t.Errorf("ID = %q", lib.ID)
	}
	if lib.DriveType != "documentLibrary" {
		t.Errorf("DriveType = %q", lib.DriveType)
	}
}

func TestAuditEntryJSON(t *testing.T) {
	raw := `{
		"action": "edit",
		"actor": "Jane Smith",
		"itemName": "report.docx",
		"occurredAt": "2025-01-15T10:30:00Z"
	}`

	var entry AuditEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if entry.Action != "edit" {
		t.Errorf("Action = %q", entry.Action)
	}
	if entry.Actor != "Jane Smith" {
		t.Errorf("Actor = %q", entry.Actor)
	}
}

func TestListSitesServer(t *testing.T) {
	sites := []Site{
		{ID: "s1", DisplayName: "HR", WebURL: "https://example.sharepoint.com/sites/hr"},
		{ID: "s2", DisplayName: "Engineering", WebURL: "https://example.sharepoint.com/sites/eng"},
	}
	respBody, _ := json.Marshal(map[string]any{"value": sites})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/sites", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result sitesResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Value) != 2 {
		t.Errorf("expected 2 sites, got %d", len(result.Value))
	}
	if result.Value[0].DisplayName != "HR" {
		t.Errorf("unexpected name: %q", result.Value[0].DisplayName)
	}
}

func TestListLibrariesServer(t *testing.T) {
	libs := []DocumentLibrary{
		{ID: "d1", DisplayName: "Documents", DriveType: "documentLibrary"},
		{ID: "d2", DisplayName: "Site Assets", DriveType: "documentLibrary"},
	}
	respBody, _ := json.Marshal(map[string]any{"value": libs})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/drives", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result drivesResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Value) != 2 {
		t.Errorf("expected 2 libraries, got %d", len(result.Value))
	}
}

func TestListLibraryFilesServer(t *testing.T) {
	items := []DriveItem{
		{ID: "f1", Name: "Budget.xlsx"},
		{ID: "f2", Name: "Proposals"},
	}
	respBody, _ := json.Marshal(map[string]any{"value": items})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/children", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result driveItemsResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Value) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Value))
	}
}

func TestOpenAndValidateFileTooLarge(t *testing.T) {
	dir := t.TempDir()
	largePath := filepath.Join(dir, "large.bin")
	f, _ := os.Create(largePath)
	buf := make([]byte, 5*1024*1024)
	f.Write(buf)
	f.Close()

	_, err := openAndValidateFile(largePath)
	if err == nil {
		t.Fatal("expected error for file > 4MB")
	}
	if !containsStr(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %s", err.Error())
	}
}

func TestOpenAndValidateFileNotExist(t *testing.T) {
	_, err := openAndValidateFile("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCreateLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "file.txt")
	f, err := createLocalFile(path)
	if err != nil {
		t.Fatalf("createLocalFile failed: %v", err)
	}
	f.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
