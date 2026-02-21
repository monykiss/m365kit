// Package send provides the kit send command for emailing documents.
package send

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/ai"
	"github.com/klytics/m365kit/internal/email"
	"github.com/klytics/m365kit/internal/formats/docx"
	"github.com/klytics/m365kit/internal/formats/pptx"
	"github.com/klytics/m365kit/internal/formats/xlsx"
)

const aiDraftSystemPrompt = "You are a professional email assistant. Based on the attached document content, write a concise email body (under 150 words). Body only — no greeting, no subject line, no sign-off."

// NewCommand returns the send command.
func NewCommand() *cobra.Command {
	var (
		to        string
		cc        string
		subject   string
		body      string
		attach    string
		aiDraft   bool
		ctxHint   string
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Email a document with optional AI-drafted body",
		Long: `Send any Office document via email with optional AI-drafted body.

Examples:
  kit send --to cfo@company.com --attach report.xlsx
  kit send --to cfo@company.com --attach report.xlsx --ai-draft
  kit send --to cfo@company.com --attach report.xlsx --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			providerName, _ := cmd.Flags().GetString("provider")
			modelName, _ := cmd.Flags().GetString("model")

			if to == "" {
				return fmt.Errorf("--to is required — specify at least one recipient email address")
			}
			if attach == "" {
				return fmt.Errorf("--attach is required — specify a file to attach")
			}

			// Parse recipients
			toList := parseEmails(to)
			ccList := parseEmails(cc)

			// Build message
			msg := email.Message{
				To:     toList,
				CC:     ccList,
				Attach: attach,
			}

			// Validate message (checks emails and attachment existence)
			if err := msg.Validate(); err != nil {
				return err
			}

			// Default subject from attachment filename
			if subject != "" {
				msg.Subject = subject
			} else {
				base := filepath.Base(attach)
				msg.Subject = strings.TrimSuffix(base, filepath.Ext(base))
			}

			// Determine body
			if aiDraft {
				drafted, err := draftBody(attach, ctxHint, providerName, modelName)
				if err != nil {
					return fmt.Errorf("AI draft failed: %w", err)
				}
				msg.Body = drafted
			} else if body != "" {
				msg.Body = body
			} else {
				msg.Body = fmt.Sprintf("Please find attached: %s", filepath.Base(attach))
			}

			// Dry-run mode
			if dryRun {
				return outputDryRun(msg, jsonFlag, aiDraft)
			}

			// Load SMTP config and send
			cfg, err := email.LoadConfig()
			if err != nil {
				return err
			}

			if err := email.Send(cfg, msg); err != nil {
				return fmt.Errorf("failed to send email: %w", err)
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"sent":       true,
					"dryRun":     false,
					"to":         msg.To,
					"subject":    msg.Subject,
					"attach":     msg.Attach,
					"attachSize": msg.AttachSize(),
					"aiDrafted":  aiDraft,
				})
			}

			fmt.Printf("Email sent to %s\n", strings.Join(msg.To, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Comma-separated recipient email addresses (required)")
	cmd.Flags().StringVar(&cc, "cc", "", "Comma-separated CC email addresses")
	cmd.Flags().StringVar(&subject, "subject", "", "Email subject (default: attachment filename)")
	cmd.Flags().StringVar(&body, "body", "", "Email body text")
	cmd.Flags().StringVar(&attach, "attach", "", "Path to file to attach (required)")
	cmd.Flags().BoolVar(&aiDraft, "ai-draft", false, "Use AI to draft the email body from the document content")
	cmd.Flags().StringVar(&ctxHint, "context", "", "Context hint for AI drafting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview email without sending")

	return cmd
}

func parseEmails(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func outputDryRun(msg email.Message, jsonFlag bool, aiDrafted bool) error {
	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"sent":       false,
			"dryRun":     true,
			"to":         msg.To,
			"cc":         msg.CC,
			"subject":    msg.Subject,
			"attach":     msg.Attach,
			"attachSize": msg.AttachSize(),
			"aiDrafted":  aiDrafted,
			"body":       msg.Body,
		})
	}

	border := color.New(color.FgHiBlack)
	label := color.New(color.FgCyan, color.Bold)
	value := color.New(color.FgWhite)

	border.Println("┌─ Email Preview ──────────────────────────────────────┐")

	printField := func(name, val string) {
		border.Print("│ ")
		label.Printf("%-8s", name+":")
		value.Printf(" %-43s", truncate(val, 43))
		border.Println(" │")
	}

	printField("To", strings.Join(msg.To, ", "))
	if len(msg.CC) > 0 {
		printField("CC", strings.Join(msg.CC, ", "))
	}
	printField("Subject", msg.Subject)
	printField("Body", msg.Body)
	if msg.Attach != "" {
		attachDesc := fmt.Sprintf("%s (%s)", filepath.Base(msg.Attach), formatSize(msg.AttachSize()))
		printField("Attach", attachDesc)
	}

	border.Println("└──────────────────────────────────────────────────────┘")

	// SMTP info
	cfg, err := email.LoadConfig()
	if err != nil {
		dim := color.New(color.FgYellow)
		dim.Println("SMTP not configured — set KIT_SMTP_* env vars to send for real")
	} else {
		fmt.Printf("Would send via %s:%d\n", cfg.Host, cfg.Port)
	}

	return nil
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%d KB", bytes/1024)
}

func draftBody(attachPath, ctxHint, providerName, modelName string) (string, error) {
	content, err := readDocumentText(attachPath)
	if err != nil {
		return "", fmt.Errorf("could not read attachment for AI drafting: %w", err)
	}

	// Truncate to 3000 chars
	if len(content) > 3000 {
		content = content[:3000] + "\n...(truncated)"
	}

	prompt := content
	if ctxHint != "" {
		prompt += "\n\nContext: " + ctxHint
	}

	provider, err := ai.NewProvider(providerName, modelName)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	result, err := provider.Infer(ctx, aiDraftSystemPrompt, []ai.Message{
		{Role: "user", Content: prompt},
	}, ai.InferOptions{MaxTokens: 512})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Content), nil
}

func readDocumentText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		doc, err := docx.ParseFile(path)
		if err != nil {
			return "", err
		}
		return doc.PlainText(), nil
	case ".xlsx":
		wb, err := xlsx.ReadFile(path)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		for i := range wb.Sheets {
			b.WriteString(fmt.Sprintf("Sheet: %s\n", wb.Sheets[i].Name))
			b.WriteString(wb.Sheets[i].ToCSV())
			b.WriteString("\n")
		}
		return b.String(), nil
	case ".pptx":
		pres, err := pptx.ReadFile(path)
		if err != nil {
			return "", err
		}
		return pres.PlainText(), nil
	default:
		// For unknown types, read as plain text
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}
