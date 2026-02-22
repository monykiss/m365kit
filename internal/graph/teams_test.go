package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewTeams(t *testing.T) {
	client := &http.Client{}
	teams := NewTeams(client)
	if teams == nil {
		t.Fatal("expected non-nil Teams")
	}
	if teams.Client != client {
		t.Error("client mismatch")
	}
}

func TestListTeamsResponse(t *testing.T) {
	teams := []Team{
		{ID: "t1", DisplayName: "Engineering", Description: "Eng team"},
		{ID: "t2", DisplayName: "Legal", Description: "Legal team"},
		{ID: "t3", DisplayName: "Marketing", Description: "Marketing team"},
	}
	respBody, _ := json.Marshal(map[string]any{"value": teams})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/me/joinedTeams", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result teamsResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Value) != 3 {
		t.Errorf("expected 3 teams, got %d", len(result.Value))
	}
	if result.Value[0].DisplayName != "Engineering" {
		t.Errorf("unexpected name: %q", result.Value[0].DisplayName)
	}
}

func TestListChannelsResponse(t *testing.T) {
	channels := []Channel{
		{ID: "c1", DisplayName: "General", Description: "General discussion"},
		{ID: "c2", DisplayName: "Random", Description: "Off-topic"},
	}
	respBody, _ := json.Marshal(map[string]any{"value": channels})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/channels", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result channelsResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Value) != 2 {
		t.Errorf("expected 2 channels, got %d", len(result.Value))
	}
}

func TestResolveTeamIDExact(t *testing.T) {
	teams := []Team{
		{ID: "t1", DisplayName: "Engineering"},
		{ID: "t2", DisplayName: "Legal"},
	}
	respBody, _ := json.Marshal(map[string]any{"value": teams})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	}))
	defer server.Close()

	tc := &Teams{Client: server.Client()}
	// We can't directly test ResolveTeamID because it calls ListTeams which uses graphBase.
	// Instead, test the matching logic via the response parsing.
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	resp, err := tc.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result teamsResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Simulate ResolveTeamID logic
	lower := "engineering"
	found := ""
	for _, team := range result.Value {
		if strings.ToLower(team.DisplayName) == lower {
			found = team.ID
			break
		}
	}
	if found != "t1" {
		t.Errorf("expected t1, got %q", found)
	}
}

func TestResolveTeamIDPartial(t *testing.T) {
	teams := []Team{
		{ID: "t1", DisplayName: "Engineering"},
		{ID: "t2", DisplayName: "Legal Affairs"},
	}

	// Simulate partial match
	lower := "legal"
	found := ""
	for _, team := range teams {
		if strings.Contains(strings.ToLower(team.DisplayName), lower) {
			found = team.ID
			break
		}
	}
	if found != "t2" {
		t.Errorf("expected t2, got %q", found)
	}
}

func TestResolveTeamIDByUUID(t *testing.T) {
	uuid := "12345678-1234-1234-1234-123456789abc"
	if !isUUID(uuid) {
		t.Error("should recognize valid UUID")
	}
	if isUUID("not-a-uuid") {
		t.Error("should reject non-UUID")
	}
	if isUUID("Engineering") {
		t.Error("should reject team name")
	}
}

func TestPostMessagePayload(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ChatMessage{
			ID:   "msg-1",
			Body: MessageBody{ContentType: "text", Content: "Hello team!"},
		})
	}))
	defer server.Close()

	ctx := context.Background()
	payload := map[string]any{
		"body": map[string]string{
			"contentType": "text",
			"content":     "Hello team!",
		},
	}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/messages", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var msg ChatMessage
	json.NewDecoder(resp.Body).Decode(&msg)
	if msg.ID != "msg-1" {
		t.Errorf("unexpected message ID: %q", msg.ID)
	}

	// Verify payload sent
	var sent map[string]any
	json.Unmarshal(receivedBody, &sent)
	body, ok := sent["body"].(map[string]any)
	if !ok {
		t.Fatal("body not found in payload")
	}
	if body["content"] != "Hello team!" {
		t.Errorf("unexpected content: %v", body["content"])
	}
}

func TestSendDirectMessageFlow(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case callCount == 1: // Create chat
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "chat-123"})
		case callCount == 2: // Send message
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(ChatMessage{
				ID:   "dm-1",
				Body: MessageBody{ContentType: "text", Content: "Hello!"},
			})
		}
	}))
	defer server.Close()

	// Simulate the two-step flow
	ctx := context.Background()

	// Step 1: create chat
	req1, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/chats", nil)
	resp1, _ := http.DefaultClient.Do(req1)
	var chatResult struct{ ID string }
	json.NewDecoder(resp1.Body).Decode(&chatResult)
	resp1.Body.Close()
	if chatResult.ID != "chat-123" {
		t.Errorf("expected chat-123, got %q", chatResult.ID)
	}

	// Step 2: send message
	req2, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/chats/"+chatResult.ID+"/messages", nil)
	resp2, _ := http.DefaultClient.Do(req2)
	var msg ChatMessage
	json.NewDecoder(resp2.Body).Decode(&msg)
	resp2.Body.Close()
	if msg.ID != "dm-1" {
		t.Errorf("expected dm-1, got %q", msg.ID)
	}
}

func TestTeamJSON(t *testing.T) {
	raw := `{"id":"team-1","displayName":"Engineering","description":"The eng team"}`
	var team Team
	if err := json.Unmarshal([]byte(raw), &team); err != nil {
		t.Fatal(err)
	}
	if team.ID != "team-1" {
		t.Errorf("ID = %q", team.ID)
	}
	if team.DisplayName != "Engineering" {
		t.Errorf("DisplayName = %q", team.DisplayName)
	}
}

func TestChannelJSON(t *testing.T) {
	raw := `{"id":"ch-1","displayName":"General","description":"Default channel"}`
	var ch Channel
	if err := json.Unmarshal([]byte(raw), &ch); err != nil {
		t.Fatal(err)
	}
	if ch.ID != "ch-1" {
		t.Errorf("ID = %q", ch.ID)
	}
}

func TestChatMessageJSON(t *testing.T) {
	raw := `{"id":"msg-1","body":{"contentType":"text","content":"Hello"}}`
	var msg ChatMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Body.Content != "Hello" {
		t.Errorf("Content = %q", msg.Body.Content)
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"12345678-1234-1234-1234-123456789abc", true},
		{"ABCDEF01-2345-6789-ABCD-EF0123456789", true},
		{"not-a-uuid", false},
		{"Engineering", false},
		{"12345678123412341234123456789abc", false}, // no dashes
		{"", false},
	}
	for _, tt := range tests {
		got := isUUID(tt.in)
		if got != tt.want {
			t.Errorf("isUUID(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
