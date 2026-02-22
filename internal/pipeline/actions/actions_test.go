package actions

import (
	"context"
	"strings"
	"testing"

	"github.com/klytics/m365kit/internal/pipeline"
)

// TestConvertActionMissingInput verifies that ConvertAction returns an error
// when no input file path is provided.
func TestConvertActionMissingInput(t *testing.T) {
	step := pipeline.Step{
		ID:     "conv1",
		Action: "convert",
		Options: map[string]string{
			"to": "md",
		},
	}

	_, err := ConvertAction(context.Background(), step, "")
	if err == nil {
		t.Fatal("expected error when input is empty")
	}
	if !strings.Contains(err.Error(), "convert requires an input file path") {
		t.Errorf("unexpected error message: %s", err)
	}
}

// TestConvertActionMissingTo verifies that ConvertAction returns an error
// when the target format (options.to) is not specified.
func TestConvertActionMissingTo(t *testing.T) {
	step := pipeline.Step{
		ID:      "conv2",
		Action:  "convert",
		Options: map[string]string{},
	}

	_, err := ConvertAction(context.Background(), step, "somefile.docx")
	if err == nil {
		t.Fatal("expected error when 'to' option is missing")
	}
	if !strings.Contains(err.Error(), "convert requires options.to") {
		t.Errorf("unexpected error message: %s", err)
	}
}

// TestOutlookInboxActionPlaceholder verifies that OutlookInboxAction returns
// an error about requiring authenticated Graph client.
func TestOutlookInboxActionPlaceholder(t *testing.T) {
	step := pipeline.Step{
		ID:     "inbox1",
		Action: "outlook.inbox",
	}

	_, err := OutlookInboxAction(context.Background(), step, "")
	if err == nil {
		t.Fatal("expected error from placeholder action")
	}
	if !strings.Contains(err.Error(), "authenticated Graph client") {
		t.Errorf("expected auth-related error, got: %s", err)
	}
	if !strings.Contains(err.Error(), "outlook.inbox") {
		t.Errorf("expected action name in error, got: %s", err)
	}
}

// TestOutlookDownloadActionPlaceholder verifies that OutlookDownloadAction
// returns an error about requiring authenticated Graph client.
func TestOutlookDownloadActionPlaceholder(t *testing.T) {
	step := pipeline.Step{
		ID:     "dl1",
		Action: "outlook.download",
	}

	_, err := OutlookDownloadAction(context.Background(), step, "")
	if err == nil {
		t.Fatal("expected error from placeholder action")
	}
	if !strings.Contains(err.Error(), "authenticated Graph client") {
		t.Errorf("expected auth-related error, got: %s", err)
	}
	if !strings.Contains(err.Error(), "outlook.download") {
		t.Errorf("expected action name in error, got: %s", err)
	}
}

// TestACLAuditActionPlaceholder verifies that ACLAuditAction returns an error
// about requiring authenticated Graph client.
func TestACLAuditActionPlaceholder(t *testing.T) {
	step := pipeline.Step{
		ID:     "acl1",
		Action: "acl.audit",
	}

	_, err := ACLAuditAction(context.Background(), step, "")
	if err == nil {
		t.Fatal("expected error from placeholder action")
	}
	if !strings.Contains(err.Error(), "authenticated Graph client") {
		t.Errorf("expected auth-related error, got: %s", err)
	}
	if !strings.Contains(err.Error(), "acl.audit") {
		t.Errorf("expected action name in error, got: %s", err)
	}
}

// TestRegisterAllActions verifies that RegisterAll registers every expected
// action name with the executor.
func TestRegisterAllActions(t *testing.T) {
	exec := pipeline.NewExecutor(false)
	RegisterAll(exec)

	// Build a pipeline with one step per expected action, each with on_failure=skip
	// so we can verify all actions are registered without needing real inputs.
	expectedActions := []string{
		"word.read",
		"word.write",
		"excel.read",
		"ai.summarize",
		"ai.analyze",
		"ai.extract",
		"email.send",
		"convert",
		"outlook.inbox",
		"outlook.download",
		"acl.audit",
	}

	p := &pipeline.Pipeline{
		Name:    "registration-test",
		Version: "1.0",
		Steps:   make([]pipeline.Step, len(expectedActions)),
	}
	for i, action := range expectedActions {
		p.Steps[i] = pipeline.Step{
			ID:        action,
			Action:    action,
			Input:     "test-input",
			OnFailure: "skip",
		}
	}

	results, err := exec.Run(context.Background(), p)
	if err != nil {
		t.Fatalf("Run should not return a fatal error with on_failure=skip: %v", err)
	}

	// Every step should have executed (not failed with "unknown action").
	// Steps may still error due to missing real files/auth, but the error
	// should NOT be about unknown actions.
	for i, r := range results {
		if r.Error != nil && strings.Contains(r.Error.Error(), "unknown action") {
			t.Errorf("action %q was not registered: %s", expectedActions[i], r.Error)
		}
	}

	if len(results) != len(expectedActions) {
		t.Errorf("expected %d step results, got %d", len(expectedActions), len(results))
	}
}
