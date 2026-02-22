// Package watch provides the "kit watch" CLI commands for file system monitoring.
package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	w "github.com/klytics/m365kit/internal/watch"
)

// NewCommand creates the "watch" command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor directories for document changes and auto-process",
		Long: `Watch directories for new or modified Office documents and trigger
automated processing based on configured rules.

Example:
  kit watch start ./contracts --ext docx --action log
  kit watch status
  kit watch stop`,
	}

	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newConfigCmd())

	return cmd
}

func newStartCmd() *cobra.Command {
	var (
		extensions []string
		recursive  bool
		actionName string
		debounce   int
	)

	cmd := &cobra.Command{
		Use:   "start <directory> [directory...]",
		Short: "Start watching directories for document changes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(extensions) == 0 {
				extensions = []string{".docx", ".xlsx", ".pptx", ".csv", ".json"}
			}

			rules := []w.Rule{
				{
					ID:         "default",
					Extensions: extensions,
					Action:     w.Action{Name: actionName, Type: actionName},
					Enabled:    true,
				},
			}

			config := w.WatchConfig{
				Directories: args,
				Rules:       rules,
				Recursive:   recursive,
				Debounce:    debounce,
			}

			watcher, err := w.New(config)
			if err != nil {
				return err
			}

			watcher.Handler = func(path string, rule w.Rule) error {
				fmt.Printf("[%s] %s → %s\n", rule.Action.Name, path, "processed")
				return nil
			}

			// Write PID
			configDir := w.DefaultConfigDir()
			if err := w.WritePIDFile(configDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write PID file: %v\n", err)
			}
			defer w.RemovePIDFile(configDir)

			// Save config for status command
			w.SaveConfig(configDir, config)

			fmt.Printf("Watching %d directory(ies) for %s files\n",
				len(args), strings.Join(extensions, ", "))
			fmt.Println("Press Ctrl+C to stop")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle signals
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				fmt.Println("\nStopping watcher...")
				cancel()
			}()

			return watcher.Start(ctx)
		},
	}

	cmd.Flags().StringSliceVar(&extensions, "ext", nil, "File extensions to watch (default: .docx,.xlsx,.pptx,.csv,.json)")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Watch directories recursively")
	cmd.Flags().StringVar(&actionName, "action", "log", "Action to perform: log, template, command")
	cmd.Flags().IntVar(&debounce, "debounce", 500, "Debounce interval in milliseconds")

	return cmd
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running watcher",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := w.DefaultConfigDir()
			pid, err := w.ReadPIDFile(configDir)
			if err != nil {
				return fmt.Errorf("no watcher running (PID file not found)")
			}

			process, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("could not find process %d: %w", pid, err)
			}

			if err := process.Signal(syscall.SIGTERM); err != nil {
				w.RemovePIDFile(configDir)
				return fmt.Errorf("could not stop watcher (PID %d): %w", pid, err)
			}

			w.RemovePIDFile(configDir)

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]any{
					"stopped": true,
					"pid":     pid,
				})
			}

			fmt.Printf("Stopped watcher (PID %d)\n", pid)
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current watcher status",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := w.DefaultConfigDir()

			pid, err := w.ReadPIDFile(configDir)
			running := err == nil

			// Check if process is actually running
			if running {
				process, err := os.FindProcess(pid)
				if err != nil {
					running = false
				} else {
					// Try sending signal 0 to check if process exists
					err = process.Signal(syscall.Signal(0))
					if err != nil {
						running = false
						w.RemovePIDFile(configDir)
					}
				}
			}

			jsonOut, _ := cmd.Flags().GetBool("json")

			if !running {
				if jsonOut {
					return json.NewEncoder(os.Stdout).Encode(map[string]any{"running": false})
				}
				fmt.Println("Watcher is not running")
				return nil
			}

			config, _ := w.LoadConfig(configDir)

			status := map[string]any{
				"running": true,
				"pid":     pid,
			}
			if config != nil {
				status["directories"] = config.Directories
				status["rules"] = len(config.Rules)
				status["recursive"] = config.Recursive
			}

			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(status)
			}

			fmt.Printf("Watcher is running (PID %d)\n", pid)
			if config != nil {
				fmt.Printf("  Directories: %s\n", strings.Join(config.Directories, ", "))
				fmt.Printf("  Rules:       %d\n", len(config.Rules))
				fmt.Printf("  Recursive:   %v\n", config.Recursive)
			}
			return nil
		},
	}
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show the current watcher configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := w.DefaultConfigDir()
			config, err := w.LoadConfig(configDir)
			if err != nil {
				return fmt.Errorf("no watcher configuration found (run 'kit watch start' first)")
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(config)
			}

			fmt.Printf("Directories: %s\n", strings.Join(config.Directories, ", "))
			fmt.Printf("Recursive:   %v\n", config.Recursive)
			fmt.Printf("Debounce:    %dms\n", config.Debounce)
			fmt.Printf("Rules:       %d\n", len(config.Rules))
			for _, r := range config.Rules {
				fmt.Printf("  [%s] ext=%v action=%s enabled=%v\n",
					r.ID, r.Extensions, r.Action.Name, r.Enabled)
			}
			return nil
		},
	}
}
