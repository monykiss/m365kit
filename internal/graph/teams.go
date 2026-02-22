package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Team represents a Microsoft Teams team.
type Team struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WebURL      string `json:"webUrl,omitempty"`
}

// Channel represents a channel within a Team.
type Channel struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WebURL      string `json:"webUrl,omitempty"`
}

// ChatMessage represents a message posted to Teams.
type ChatMessage struct {
	ID        string      `json:"id"`
	Body      MessageBody `json:"body"`
	CreatedAt time.Time   `json:"createdDateTime"`
	WebURL    string      `json:"webUrl,omitempty"`
}

// MessageBody holds the content of a chat message.
type MessageBody struct {
	ContentType string `json:"contentType"` // "text" or "html"
	Content     string `json:"content"`
}

type teamsResponse struct {
	Value []Team `json:"value"`
}

type channelsResponse struct {
	Value []Channel `json:"value"`
}

// Teams provides operations on Microsoft Teams.
type Teams struct {
	Client *http.Client
}

// NewTeams creates a new Teams client with an authenticated HTTP client.
func NewTeams(client *http.Client) *Teams {
	return &Teams{Client: client}
}

// ListTeams returns all Teams the user is a member of.
func (t *Teams) ListTeams(ctx context.Context) ([]Team, error) {
	endpoint := graphBase + "/me/joinedTeams"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Teams list request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Teams API returned %d: %s", resp.StatusCode, string(body))
	}

	var result teamsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse Teams response: %w", err)
	}

	return result.Value, nil
}

// ListChannels returns channels in a team.
func (t *Teams) ListChannels(ctx context.Context, teamID string) ([]Channel, error) {
	endpoint := graphBase + "/teams/" + teamID + "/channels"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Teams channels request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Teams API returned %d: %s", resp.StatusCode, string(body))
	}

	var result channelsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse channels response: %w", err)
	}

	return result.Value, nil
}

// ResolveTeamID looks up a team by display name (case-insensitive, partial match).
// If nameOrID looks like a UUID, returns it directly.
func (t *Teams) ResolveTeamID(ctx context.Context, nameOrID string) (string, error) {
	if isUUID(nameOrID) {
		return nameOrID, nil
	}

	teams, err := t.ListTeams(ctx)
	if err != nil {
		return "", err
	}

	lower := strings.ToLower(nameOrID)
	for _, team := range teams {
		if strings.ToLower(team.DisplayName) == lower {
			return team.ID, nil
		}
	}
	// Partial match
	for _, team := range teams {
		if strings.Contains(strings.ToLower(team.DisplayName), lower) {
			return team.ID, nil
		}
	}

	return "", fmt.Errorf("team %q not found — run: kit teams list", nameOrID)
}

// ResolveChannelID looks up a channel by display name within a team.
func (t *Teams) ResolveChannelID(ctx context.Context, teamID, nameOrID string) (string, error) {
	if isUUID(nameOrID) {
		return nameOrID, nil
	}

	channels, err := t.ListChannels(ctx, teamID)
	if err != nil {
		return "", err
	}

	// Strip leading # if present
	name := strings.TrimPrefix(nameOrID, "#")
	lower := strings.ToLower(name)

	for _, ch := range channels {
		if strings.ToLower(ch.DisplayName) == lower {
			return ch.ID, nil
		}
	}
	for _, ch := range channels {
		if strings.Contains(strings.ToLower(ch.DisplayName), lower) {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel %q not found in team — run: kit teams channels --team %s", nameOrID, teamID)
}

// PostMessage sends a text message to a channel.
func (t *Teams) PostMessage(ctx context.Context, teamID, channelID, text string) (*ChatMessage, error) {
	endpoint := graphBase + "/teams/" + teamID + "/channels/" + channelID + "/messages"

	payload := map[string]any{
		"body": map[string]string{
			"contentType": "text",
			"content":     text,
		},
	}
	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post message failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("post message failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var msg ChatMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("could not parse message response: %w", err)
	}

	return &msg, nil
}

// PostMessageWithFile uploads a file to the channel's Files tab and posts a message referencing it.
func (t *Teams) PostMessageWithFile(ctx context.Context, teamID, channelID, message, filePath string) (*ChatMessage, error) {
	// Step 1: Upload file to the team's drive
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat file: %w", err)
	}
	if info.Size() > 4*1024*1024 {
		return nil, fmt.Errorf("file too large for upload (%d bytes, max 4MB)", info.Size())
	}

	fileName := info.Name()
	uploadEndpoint := graphBase + "/teams/" + teamID + "/drive/root:/" + url.PathEscape(fileName) + ":/content"

	uploadReq, err := http.NewRequestWithContext(ctx, "PUT", uploadEndpoint, f)
	if err != nil {
		return nil, err
	}
	uploadReq.Header.Set("Content-Type", "application/octet-stream")

	uploadResp, err := t.Client.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("file upload failed: %w", err)
	}
	defer uploadResp.Body.Close()

	uploadBody, _ := io.ReadAll(uploadResp.Body)
	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("file upload failed (HTTP %d): %s", uploadResp.StatusCode, string(uploadBody))
	}

	var uploadResult struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		WebURL string `json:"webUrl"`
		ETag   string `json:"eTag"`
	}
	json.Unmarshal(uploadBody, &uploadResult)

	// Step 2: Post message with file reference
	if message == "" {
		message = "Shared: " + fileName
	}

	htmlContent := fmt.Sprintf(`%s<br><a href="%s">%s</a>`, message, uploadResult.WebURL, fileName)
	payload := map[string]any{
		"body": map[string]string{
			"contentType": "html",
			"content":     htmlContent,
		},
	}
	jsonData, _ := json.Marshal(payload)

	endpoint := graphBase + "/teams/" + teamID + "/channels/" + channelID + "/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post message failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("post message failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var msg ChatMessage
	json.Unmarshal(body, &msg)
	return &msg, nil
}

// SendDirectMessage sends a DM to a user by email address.
func (t *Teams) SendDirectMessage(ctx context.Context, toEmail, message string) (*ChatMessage, error) {
	// Step 1: Create or get 1:1 chat
	chatPayload := map[string]any{
		"chatType": "oneOnOne",
		"members": []map[string]any{
			{
				"@odata.type":     "#microsoft.graph.aadUserConversationMember",
				"roles":           []string{"owner"},
				"user@odata.bind": graphBase + "/users/" + url.PathEscape(toEmail),
			},
		},
	}
	chatJSON, _ := json.Marshal(chatPayload)

	chatReq, err := http.NewRequestWithContext(ctx, "POST", graphBase+"/chats", bytes.NewReader(chatJSON))
	if err != nil {
		return nil, err
	}
	chatReq.Header.Set("Content-Type", "application/json")

	chatResp, err := t.Client.Do(chatReq)
	if err != nil {
		return nil, fmt.Errorf("create chat failed: %w", err)
	}
	defer chatResp.Body.Close()

	chatBody, _ := io.ReadAll(chatResp.Body)
	if chatResp.StatusCode != http.StatusCreated && chatResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create chat failed (HTTP %d): %s", chatResp.StatusCode, string(chatBody))
	}

	var chatResult struct {
		ID string `json:"id"`
	}
	json.Unmarshal(chatBody, &chatResult)

	// Step 2: Send message to chat
	msgPayload := map[string]any{
		"body": map[string]string{
			"contentType": "text",
			"content":     message,
		},
	}
	msgJSON, _ := json.Marshal(msgPayload)

	msgReq, err := http.NewRequestWithContext(ctx, "POST", graphBase+"/chats/"+chatResult.ID+"/messages", bytes.NewReader(msgJSON))
	if err != nil {
		return nil, err
	}
	msgReq.Header.Set("Content-Type", "application/json")

	msgResp, err := t.Client.Do(msgReq)
	if err != nil {
		return nil, fmt.Errorf("send DM failed: %w", err)
	}
	defer msgResp.Body.Close()

	msgBody, _ := io.ReadAll(msgResp.Body)
	if msgResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("send DM failed (HTTP %d): %s", msgResp.StatusCode, string(msgBody))
	}

	var msg ChatMessage
	json.Unmarshal(msgBody, &msg)
	return &msg, nil
}

// isUUID checks if a string looks like a UUID.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
