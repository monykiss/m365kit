// Package shell provides the interactive M365Kit REPL.
package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

// CommandRunner executes a kit command and returns its output.
// This is set by the cmd/shell package to avoid import cycles.
type CommandRunner func(ctx context.Context, args []string, stdout, stderr io.Writer) error

// DefaultRunner is the command runner used by the shell session.
var DefaultRunner CommandRunner

// Session manages an interactive kit shell session.
type Session struct {
	DefaultSite    string
	DefaultTeam    string
	LastOutput     string
	CommandHistory []string
	HistoryFile    string
	StartTime      time.Time

	// KnownCommands is the list of top-level commands for completion.
	KnownCommands []string
}

// NewSession creates a new interactive session.
func NewSession() (*Session, error) {
	home, _ := os.UserHomeDir()
	histFile := filepath.Join(home, ".kit", "shell_history")

	// Ensure parent dir exists
	os.MkdirAll(filepath.Dir(histFile), 0755)

	return &Session{
		HistoryFile: histFile,
		StartTime:   time.Now(),
		KnownCommands: []string{
			"word", "excel", "pptx", "ai", "pipeline", "batch",
			"auth", "onedrive", "sharepoint", "teams", "outlook", "acl",
			"fs", "template", "report", "watch",
			"send", "diff", "convert",
			"config", "completion", "update", "doctor", "version",
			"org", "audit", "admin", "plugin", "shell",
			"help", "exit", "quit", "history", "set",
		},
	}, nil
}

// Run starts the REPL loop. Blocks until 'exit' or Ctrl+D.
func (s *Session) Run(ctx context.Context) error {
	if DefaultRunner == nil {
		return fmt.Errorf("shell runner not configured")
	}

	completer := readline.NewPrefixCompleter(s.buildCompleter()...)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "kit> ",
		HistoryFile:     s.HistoryFile,
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	fmt.Printf("M365Kit — Interactive Shell\n")
	fmt.Println("Type 'help' for commands, 'exit' to quit.")
	fmt.Println()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF or interrupt
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		s.CommandHistory = append(s.CommandHistory, line)

		switch {
		case line == "exit" || line == "quit":
			elapsed := time.Since(s.StartTime)
			fmt.Printf("\nSession ended. %d commands run in %s.\n",
				len(s.CommandHistory)-1, formatDuration(elapsed))
			return nil
		case line == "help":
			s.printHelp()
		case line == "history":
			for i, cmd := range s.CommandHistory {
				fmt.Printf("  %d  %s\n", i+1, cmd)
			}
		case strings.HasPrefix(line, "set site "):
			s.DefaultSite = strings.TrimPrefix(line, "set site ")
			fmt.Printf("Default SharePoint site: %s\n", s.DefaultSite)
		case strings.HasPrefix(line, "set team "):
			s.DefaultTeam = strings.TrimPrefix(line, "set team ")
			fmt.Printf("Default team: %s\n", s.DefaultTeam)
		case strings.HasPrefix(line, "-- "):
			// Raw shell passthrough — not supported in this implementation
			fmt.Println("Shell passthrough not supported. Use standard kit commands.")
		default:
			output, err := s.Eval(ctx, line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			} else if output != "" {
				fmt.Print(output)
				if !strings.HasSuffix(output, "\n") {
					fmt.Println()
				}
			}
		}
	}

	return nil
}

// Eval runs a single command string and returns its output.
func (s *Session) Eval(ctx context.Context, command string) (string, error) {
	if DefaultRunner == nil {
		return "", fmt.Errorf("shell runner not configured")
	}

	args := strings.Fields(command)
	if len(args) == 0 {
		return "", nil
	}

	var stdout, stderr bytes.Buffer
	err := DefaultRunner(ctx, args, &stdout, &stderr)

	output := stdout.String()
	s.LastOutput = output

	if errOut := stderr.String(); errOut != "" && err != nil {
		return output, fmt.Errorf("%s", strings.TrimSpace(errOut))
	}

	return output, err
}

// Complete returns tab-completion candidates for the given input.
func (s *Session) Complete(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return s.KnownCommands
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return s.KnownCommands
	}

	// Complete top-level command
	if len(parts) == 1 && !strings.HasSuffix(input, " ") {
		prefix := parts[0]
		var matches []string
		for _, cmd := range s.KnownCommands {
			if strings.HasPrefix(cmd, prefix) {
				matches = append(matches, cmd)
			}
		}
		sort.Strings(matches)
		return matches
	}

	// For subcommands, return common subcommands based on parent
	parent := parts[0]
	subcommands := s.subcommandsFor(parent)
	if len(parts) == 2 && !strings.HasSuffix(input, " ") {
		prefix := parts[1]
		var matches []string
		for _, sub := range subcommands {
			if strings.HasPrefix(sub, prefix) {
				matches = append(matches, sub)
			}
		}
		return matches
	}

	// For flags
	if strings.HasSuffix(input, " -") || (len(parts) > 0 && strings.HasPrefix(parts[len(parts)-1], "-")) {
		return []string{"--json", "--verbose", "--help", "--output"}
	}

	return nil
}

func (s *Session) subcommandsFor(parent string) []string {
	subs := map[string][]string{
		"word":       {"read", "write", "edit"},
		"excel":      {"read", "write", "analyze"},
		"pptx":       {"read", "generate"},
		"ai":         {"summarize", "analyze", "extract", "ask"},
		"auth":       {"login", "whoami", "status", "logout"},
		"onedrive":   {"ls", "get", "put", "recent", "search", "share"},
		"sharepoint": {"sites", "libs", "ls", "get", "put", "audit"},
		"teams":      {"list", "channels", "post", "dm"},
		"outlook":    {"inbox", "read", "download", "reply"},
		"acl":        {"audit", "external", "broken", "users"},
		"fs":         {"scan", "rename", "dedupe", "stale", "organize", "manifest"},
		"template":   {"list", "show", "apply", "add", "vars"},
		"report":     {"generate", "preview"},
		"watch":      {"start", "stop", "status"},
		"config":     {"init", "show", "set", "validate"},
		"org":        {"show", "validate", "init", "status"},
		"audit":      {"log", "clear", "status"},
		"admin":      {"stats", "users", "telemetry"},
		"plugin":     {"list", "install", "remove", "run", "show", "new"},
	}
	return subs[parent]
}

func (s *Session) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println()
	fmt.Println("  Documents:  word, excel, pptx, convert, diff")
	fmt.Println("  AI:         ai summarize/analyze/extract/ask")
	fmt.Println("  Cloud:      auth, onedrive, sharepoint, teams, outlook, acl")
	fmt.Println("  Files:      fs, template, report, batch, pipeline")
	fmt.Println("  Admin:      org, audit, admin, config, plugin")
	fmt.Println("  System:     doctor, version, update")
	fmt.Println()
	fmt.Println("Shell commands:")
	fmt.Println("  help       — show this help")
	fmt.Println("  history    — show command history")
	fmt.Println("  set site <url> — set default SharePoint site")
	fmt.Println("  set team <name> — set default Teams team")
	fmt.Println("  exit       — exit the shell")
}

func (s *Session) buildCompleter() []readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface
	for _, cmd := range s.KnownCommands {
		subs := s.subcommandsFor(cmd)
		if len(subs) > 0 {
			var subItems []readline.PrefixCompleterInterface
			for _, sub := range subs {
				subItems = append(subItems, readline.PcItem(sub))
			}
			items = append(items, readline.PcItem(cmd, subItems...))
		} else {
			items = append(items, readline.PcItem(cmd))
		}
	}
	return items
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
