package graph

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListInboxResponse(t *testing.T) {
	messages := []EmailMessage{
		{ID: "m1", Subject: "Hello", IsRead: false, HasAttachments: true},
		{ID: "m2", Subject: "Meeting notes", IsRead: true},
		{ID: "m3", Subject: "Q4 Report", IsRead: false},
		{ID: "m4", Subject: "FYI", IsRead: true},
		{ID: "m5", Subject: "Action needed", IsRead: false},
	}
	respBody, _ := json.Marshal(map[string]any{"value": messages})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/me/messages", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result messagesResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Value) != 5 {
		t.Errorf("expected 5 messages, got %d", len(result.Value))
	}
}

func TestListInboxFromFilter(t *testing.T) {
	var receivedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use RawQuery decoded to check the filter
		decoded, _ := url.QueryUnescape(r.URL.String())
		receivedURL = decoded
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{Value: []EmailMessage{}})
	}))
	defer server.Close()

	o := &Outlook{Client: &http.Client{Transport: &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}}}
	ctx := context.Background()
	o.ListInbox(ctx, InboxFilter{From: "alice@test.com", Limit: 10})

	if !strings.Contains(receivedURL, "from/emailAddress/address") {
		t.Errorf("expected from filter in URL, got: %s", receivedURL)
	}
}

func TestListInboxHasAttachmentFilter(t *testing.T) {
	var receivedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded, _ := url.QueryUnescape(r.URL.String())
		receivedURL = decoded
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesResponse{Value: []EmailMessage{}})
	}))
	defer server.Close()

	o := &Outlook{Client: &http.Client{Transport: &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}}}
	ctx := context.Background()
	o.ListInbox(ctx, InboxFilter{HasAttachment: true})

	if !strings.Contains(receivedURL, "hasAttachments") {
		t.Errorf("expected hasAttachments filter in URL, got: %s", receivedURL)
	}
}

func TestGetMessageByIndex(t *testing.T) {
	messages := []EmailMessage{
		{ID: "m1", Subject: "First"},
		{ID: "m2", Subject: "Second"},
		{ID: "m3", Subject: "Third"},
	}
	respBody, _ := json.Marshal(map[string]any{"value": messages})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	o := &Outlook{Client: &http.Client{Transport: &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}}}
	ctx := context.Background()

	msg, err := o.GetMessageByIndex(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Subject != "Second" {
		t.Errorf("expected 'Second', got %q", msg.Subject)
	}
}

func TestListAttachmentsResponse(t *testing.T) {
	attachments := []Attachment{
		{ID: "a1", Name: "report.docx", ContentType: "application/docx", Size: 1024},
		{ID: "a2", Name: "image.png", ContentType: "image/png", Size: 2048, IsInline: true},
	}
	respBody, _ := json.Marshal(map[string]any{"value": attachments})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	o := &Outlook{Client: &http.Client{Transport: &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}}}
	ctx := context.Background()

	atts, err := o.ListAttachments(ctx, "msg-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(atts))
	}
	if atts[0].Name != "report.docx" {
		t.Errorf("expected report.docx, got %q", atts[0].Name)
	}
}

func TestDownloadAttachmentWritesFile(t *testing.T) {
	content := []byte("Hello, this is a test file.")
	encoded := base64.StdEncoding.EncodeToString(content)

	att := Attachment{
		ID:           "a1",
		Name:         "test.docx",
		ContentBytes: encoded,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(att)
	}))
	defer server.Close()

	o := &Outlook{Client: &http.Client{Transport: &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}}}
	ctx := context.Background()

	destDir := t.TempDir()
	path, err := o.DownloadAttachment(ctx, "msg-1", "a1", destDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(destDir, "test.docx")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Hello, this is a test file." {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestMarkAsReadRequest(t *testing.T) {
	var method string
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"isRead": true})
	}))
	defer server.Close()

	o := &Outlook{Client: &http.Client{Transport: &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}}}
	ctx := context.Background()

	if err := o.MarkAsRead(ctx, "msg-1"); err != nil {
		t.Fatal(err)
	}

	if method != "PATCH" {
		t.Errorf("expected PATCH, got %s", method)
	}
	if !strings.Contains(string(receivedBody), `"isRead"`) {
		t.Errorf("expected isRead in body, got: %s", string(receivedBody))
	}
}

func TestReplyRequest(t *testing.T) {
	var method string
	var receivedURL string
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		receivedURL = r.URL.String()
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	o := &Outlook{Client: &http.Client{Transport: &rewriteTransport{base: server.URL, wrapped: http.DefaultTransport}}}
	ctx := context.Background()

	if err := o.Reply(ctx, "msg-1", "Thanks for the update!"); err != nil {
		t.Fatal(err)
	}

	if method != "POST" {
		t.Errorf("expected POST, got %s", method)
	}
	if !strings.Contains(receivedURL, "/reply") {
		t.Errorf("expected /reply in URL, got: %s", receivedURL)
	}
	if !strings.Contains(string(receivedBody), "Thanks for the update!") {
		t.Errorf("expected reply body, got: %s", string(receivedBody))
	}
}

func TestIsOfficeAttachment(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"report.docx", true},
		{"data.xlsx", true},
		{"slides.pptx", true},
		{"contract.pdf", true},
		{"image.png", false},
		{"readme.txt", false},
		{"photo.jpg", false},
	}
	for _, tt := range tests {
		got := IsOfficeAttachment(tt.name)
		if got != tt.want {
			t.Errorf("IsOfficeAttachment(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNewOutlook(t *testing.T) {
	client := &http.Client{}
	o := NewOutlook(client)
	if o == nil {
		t.Fatal("expected non-nil Outlook")
	}
	if o.Client != client {
		t.Error("client mismatch")
	}
}

func TestEmailMessageJSON(t *testing.T) {
	raw := `{"id":"m1","subject":"Test","from":{"emailAddress":{"name":"Alice","address":"alice@test.com"}},"receivedDateTime":"2026-02-21T09:00:00Z","isRead":false,"hasAttachments":true}`
	var msg EmailMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.ID != "m1" {
		t.Errorf("ID = %q", msg.ID)
	}
	if msg.From.EmailAddress.Address != "alice@test.com" {
		t.Errorf("From = %q", msg.From.EmailAddress.Address)
	}
	if !msg.HasAttachments {
		t.Error("expected hasAttachments=true")
	}
}

func TestFormatEmailDate(t *testing.T) {
	ts := time.Date(2026, 2, 21, 9, 14, 0, 0, time.UTC)
	got := FormatEmailDate(ts)
	if got != "2026-02-21 09:14" {
		t.Errorf("FormatEmailDate = %q", got)
	}

	zero := FormatEmailDate(time.Time{})
	if zero != "" {
		t.Errorf("expected empty for zero time, got %q", zero)
	}
}

// rewriteTransport rewrites the host in requests to point to the test server.
type rewriteTransport struct {
	base    string
	wrapped http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace graph.microsoft.com with test server
	newURL := t.base + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	newReq, _ := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	return t.wrapped.RoundTrip(newReq)
}
