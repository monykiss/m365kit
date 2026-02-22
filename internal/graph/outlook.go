package graph

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EmailMessage represents an Outlook email message.
type EmailMessage struct {
	ID             string         `json:"id"`
	Subject        string         `json:"subject"`
	From           EmailRecipient `json:"from"`
	To             []EmailRecipient `json:"toRecipients"`
	Body           EmailBody      `json:"body"`
	ReceivedAt     time.Time      `json:"receivedDateTime"`
	IsRead         bool           `json:"isRead"`
	HasAttachments bool           `json:"hasAttachments"`
	WebLink        string         `json:"webLink,omitempty"`
}

// EmailRecipient holds an email address with display name.
type EmailRecipient struct {
	EmailAddress EmailAddr `json:"emailAddress"`
}

// EmailAddr holds the address and name.
type EmailAddr struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// EmailBody holds the body content of an email.
type EmailBody struct {
	ContentType string `json:"contentType"` // "text" or "html"
	Content     string `json:"content"`
}

// Attachment represents an email attachment.
type Attachment struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ContentType    string `json:"contentType"`
	Size           int64  `json:"size"`
	IsInline       bool   `json:"isInline"`
	ContentBytes   string `json:"contentBytes,omitempty"` // base64 encoded
}

// InboxFilter configures which emails to retrieve.
type InboxFilter struct {
	From          string
	Subject       string
	HasAttachment bool
	UnreadOnly    bool
	Since         time.Time
	Limit         int
}

// Outlook provides Microsoft Outlook operations via Graph API.
type Outlook struct {
	Client *http.Client
}

// NewOutlook creates a new Outlook client.
func NewOutlook(client *http.Client) *Outlook {
	return &Outlook{Client: client}
}

type messagesResponse struct {
	Value []EmailMessage `json:"value"`
}

type attachmentsResponse struct {
	Value []Attachment `json:"value"`
}

// ListInbox returns recent emails with optional filters.
func (o *Outlook) ListInbox(ctx context.Context, filter InboxFilter) ([]EmailMessage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	params := url.Values{}
	params.Set("$top", fmt.Sprintf("%d", limit))
	params.Set("$orderby", "receivedDateTime desc")
	params.Set("$select", "id,subject,from,toRecipients,receivedDateTime,isRead,hasAttachments,webLink")

	// Build OData filter
	var filters []string
	if filter.From != "" {
		filters = append(filters, fmt.Sprintf("from/emailAddress/address eq '%s'", filter.From))
	}
	if filter.Subject != "" {
		filters = append(filters, fmt.Sprintf("contains(subject, '%s')", filter.Subject))
	}
	if filter.HasAttachment {
		filters = append(filters, "hasAttachments eq true")
	}
	if filter.UnreadOnly {
		filters = append(filters, "isRead eq false")
	}
	if !filter.Since.IsZero() {
		filters = append(filters, fmt.Sprintf("receivedDateTime ge %s", filter.Since.Format(time.RFC3339)))
	}
	if len(filters) > 0 {
		params.Set("$filter", strings.Join(filters, " and "))
	}

	endpoint := graphBase + "/me/messages?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not list inbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("inbox request failed (%d): %s", resp.StatusCode, string(body))
	}

	var result messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("could not parse inbox response: %w", err)
	}
	return result.Value, nil
}

// GetMessage retrieves a single email by ID.
func (o *Outlook) GetMessage(ctx context.Context, id string) (*EmailMessage, error) {
	endpoint := graphBase + "/me/messages/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get message failed (%d): %s", resp.StatusCode, string(body))
	}

	var msg EmailMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, fmt.Errorf("could not parse message: %w", err)
	}
	return &msg, nil
}

// GetMessageByIndex retrieves the Nth message from inbox (1-indexed).
func (o *Outlook) GetMessageByIndex(ctx context.Context, n int) (*EmailMessage, error) {
	if n < 1 {
		return nil, fmt.Errorf("message index must be >= 1, got %d", n)
	}
	messages, err := o.ListInbox(ctx, InboxFilter{Limit: n})
	if err != nil {
		return nil, err
	}
	if n > len(messages) {
		return nil, fmt.Errorf("message index %d out of range (inbox has %d messages)", n, len(messages))
	}
	return &messages[n-1], nil
}

// ListAttachments returns attachments for a message.
func (o *Outlook) ListAttachments(ctx context.Context, messageID string) ([]Attachment, error) {
	endpoint := graphBase + "/me/messages/" + url.PathEscape(messageID) + "/attachments"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not list attachments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list attachments failed (%d): %s", resp.StatusCode, string(body))
	}

	var result attachmentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("could not parse attachments: %w", err)
	}
	return result.Value, nil
}

// DownloadAttachment downloads an attachment to a local directory.
// Returns the local file path written.
func (o *Outlook) DownloadAttachment(ctx context.Context, messageID, attachmentID, destDir string) (string, error) {
	endpoint := graphBase + "/me/messages/" + url.PathEscape(messageID) + "/attachments/" + url.PathEscape(attachmentID)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not download attachment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download attachment failed (%d): %s", resp.StatusCode, string(body))
	}

	var att Attachment
	if err := json.NewDecoder(resp.Body).Decode(&att); err != nil {
		return "", fmt.Errorf("could not parse attachment: %w", err)
	}

	if att.ContentBytes == "" {
		return "", fmt.Errorf("attachment %s has no content", att.Name)
	}

	decoded, err := base64.StdEncoding.DecodeString(att.ContentBytes)
	if err != nil {
		return "", fmt.Errorf("could not decode attachment content: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("could not create output directory: %w", err)
	}

	outPath := filepath.Join(destDir, att.Name)
	if err := os.WriteFile(outPath, decoded, 0644); err != nil {
		return "", fmt.Errorf("could not write attachment: %w", err)
	}
	return outPath, nil
}

// MarkAsRead marks a message as read.
func (o *Outlook) MarkAsRead(ctx context.Context, messageID string) error {
	endpoint := graphBase + "/me/messages/" + url.PathEscape(messageID)
	body := []byte(`{"isRead": true}`)
	req, err := http.NewRequestWithContext(ctx, "PATCH", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return fmt.Errorf("could not mark as read: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mark as read failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// Reply sends a reply to a message.
func (o *Outlook) Reply(ctx context.Context, messageID, bodyText string) error {
	endpoint := graphBase + "/me/messages/" + url.PathEscape(messageID) + "/reply"
	payload := map[string]any{
		"message": map[string]any{
			"body": map[string]string{
				"contentType": "text",
				"content":     bodyText,
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return fmt.Errorf("could not reply: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reply failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// IsOfficeAttachment returns true if the attachment is an Office document.
func IsOfficeAttachment(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".docx", ".xlsx", ".pptx", ".doc", ".xls", ".ppt", ".pdf":
		return true
	}
	return false
}

// FormatEmailDate formats an email timestamp for display.
func FormatEmailDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}
