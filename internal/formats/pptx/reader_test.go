package pptx

import (
	"testing"
)

func TestParseInvalidData(t *testing.T) {
	_, err := Parse([]byte("not a zip file"))
	if err == nil {
		t.Fatal("expected error for invalid data")
	}
}

func TestPlainText(t *testing.T) {
	pres := &Presentation{
		Slides: []Slide{
			{
				Number:      1,
				Title:       "Introduction",
				TextContent: []string{"Introduction", "Welcome to the presentation"},
			},
			{
				Number:      2,
				Title:       "Overview",
				TextContent: []string{"Overview", "Key points here"},
			},
		},
	}

	text := pres.PlainText()
	if text == "" {
		t.Fatal("PlainText returned empty string")
	}

	if !containsStr(text, "Slide 1") {
		t.Error("missing slide 1 header")
	}
	if !containsStr(text, "Introduction") {
		t.Error("missing slide 1 title")
	}
	if !containsStr(text, "Welcome to the presentation") {
		t.Error("missing slide 1 content")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
