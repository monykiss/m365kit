// Package doctor provides the "kit doctor" command for checking system health.
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Check represents a single health check result.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warning", "error"
	Message string `json:"message"`
}

// NewCommand creates the "doctor" command.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system health and dependencies",
		Long:  "Run diagnostic checks to verify M365Kit is properly configured.",
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := runChecks()

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(checks)
			}

			green := color.New(color.FgGreen).SprintFunc()
			yellow := color.New(color.FgYellow).SprintFunc()
			red := color.New(color.FgRed).SprintFunc()

			fmt.Println("M365Kit Doctor")
			fmt.Println("==============")
			fmt.Println()

			okCount, warnCount, errCount := 0, 0, 0
			for _, c := range checks {
				var icon string
				switch c.Status {
				case "ok":
					icon = green("✓")
					okCount++
				case "warning":
					icon = yellow("!")
					warnCount++
				case "error":
					icon = red("✗")
					errCount++
				}
				fmt.Printf("  %s %s: %s\n", icon, c.Name, c.Message)
			}

			fmt.Println()
			fmt.Printf("  %d passed, %d warnings, %d errors\n", okCount, warnCount, errCount)

			if errCount > 0 {
				return fmt.Errorf("%d check(s) failed", errCount)
			}
			return nil
		},
	}
}

func runChecks() []Check {
	var checks []Check

	// Check Go runtime
	checks = append(checks, Check{
		Name:    "Go Runtime",
		Status:  "ok",
		Message: fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	})

	// Check config directory
	home, _ := os.UserHomeDir()
	configDir := home + "/.kit"
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		checks = append(checks, Check{
			Name:    "Config Directory",
			Status:  "ok",
			Message: configDir,
		})
	} else {
		checks = append(checks, Check{
			Name:    "Config Directory",
			Status:  "warning",
			Message: fmt.Sprintf("%s not found — run 'kit config init'", configDir),
		})
	}

	// Check config file
	configFile := configDir + "/config.yaml"
	if _, err := os.Stat(configFile); err == nil {
		checks = append(checks, Check{
			Name:    "Config File",
			Status:  "ok",
			Message: configFile,
		})
	} else {
		checks = append(checks, Check{
			Name:    "Config File",
			Status:  "warning",
			Message: "Not found — run 'kit config init'",
		})
	}

	// Check auth token
	tokenFile := configDir + "/token.json"
	if _, err := os.Stat(tokenFile); err == nil {
		checks = append(checks, Check{
			Name:    "Auth Token",
			Status:  "ok",
			Message: "Token file exists",
		})
	} else {
		checks = append(checks, Check{
			Name:    "Auth Token",
			Status:  "warning",
			Message: "Not authenticated — run 'kit auth login' for M365 features",
		})
	}

	// Check AI provider
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		checks = append(checks, Check{
			Name:    "AI Provider (Anthropic)",
			Status:  "ok",
			Message: "ANTHROPIC_API_KEY set",
		})
	} else if os.Getenv("OPENAI_API_KEY") != "" {
		checks = append(checks, Check{
			Name:    "AI Provider (OpenAI)",
			Status:  "ok",
			Message: "OPENAI_API_KEY set",
		})
	} else {
		// Check if ollama is available
		if _, err := exec.LookPath("ollama"); err == nil {
			checks = append(checks, Check{
				Name:    "AI Provider (Ollama)",
				Status:  "ok",
				Message: "Ollama found in PATH",
			})
		} else {
			checks = append(checks, Check{
				Name:    "AI Provider",
				Status:  "warning",
				Message: "No API key set — set ANTHROPIC_API_KEY or OPENAI_API_KEY for AI features",
			})
		}
	}

	// Check Azure client ID
	if os.Getenv("KIT_AZURE_CLIENT_ID") != "" {
		checks = append(checks, Check{
			Name:    "Azure Client ID",
			Status:  "ok",
			Message: "KIT_AZURE_CLIENT_ID set",
		})
	} else {
		checks = append(checks, Check{
			Name:    "Azure Client ID",
			Status:  "warning",
			Message: "KIT_AZURE_CLIENT_ID not set — required for M365 features",
		})
	}

	// Check git
	if _, err := exec.LookPath("git"); err == nil {
		checks = append(checks, Check{
			Name:    "Git",
			Status:  "ok",
			Message: "Available",
		})
	} else {
		checks = append(checks, Check{
			Name:    "Git",
			Status:  "warning",
			Message: "Not found in PATH",
		})
	}

	return checks
}
