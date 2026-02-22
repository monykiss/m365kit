// Package teams provides CLI commands for Microsoft Teams integration.
package teams

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/auth"
	"github.com/klytics/m365kit/internal/graph"
)

// NewCommand returns the teams command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teams",
		Short: "Microsoft Teams messaging and file sharing",
		Long:  "List teams, post messages, share files, and send DMs via Microsoft Teams.",
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newChannelsCommand())
	cmd.AddCommand(newPostCommand())
	cmd.AddCommand(newShareCommand())
	cmd.AddCommand(newDMCommand())

	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your Teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			tc := graph.NewTeams(client)
			teams, err := tc.ListTeams(ctx)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(teams)
			}

			if len(teams) == 0 {
				fmt.Println("No teams found")
				return nil
			}

			fmt.Printf("Your Teams (%d)\n\n", len(teams))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tID\tDESCRIPTION\n")
			for _, t := range teams {
				desc := t.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", t.DisplayName, t.ID, desc)
			}
			w.Flush()
			fmt.Printf("\n%d teams\n", len(teams))
			return nil
		},
	}
}

func newChannelsCommand() *cobra.Command {
	var teamName string
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "List channels in a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			if teamName == "" {
				return fmt.Errorf("--team is required")
			}

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			tc := graph.NewTeams(client)
			teamID, err := tc.ResolveTeamID(ctx, teamName)
			if err != nil {
				return err
			}

			channels, err := tc.ListChannels(ctx, teamID)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(channels)
			}

			if len(channels) == 0 {
				fmt.Println("No channels found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tID\tDESCRIPTION\n")
			for _, ch := range channels {
				fmt.Fprintf(w, "#%s\t%s\t%s\n", ch.DisplayName, ch.ID, ch.Description)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&teamName, "team", "", "Team name or ID (required)")
	return cmd
}

func newPostCommand() *cobra.Command {
	var (
		teamName    string
		channelName string
		message     string
		attachFile  string
		useStdin    bool
		dryRun      bool
	)
	cmd := &cobra.Command{
		Use:   "post",
		Short: "Post a message to a Teams channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			if teamName == "" {
				return fmt.Errorf("--team is required")
			}
			if channelName == "" {
				return fmt.Errorf("--channel is required")
			}

			// Read message from stdin if --stdin
			if useStdin {
				scanner := bufio.NewScanner(os.Stdin)
				var lines []byte
				for scanner.Scan() {
					if len(lines) > 0 {
						lines = append(lines, '\n')
					}
					lines = append(lines, scanner.Bytes()...)
				}
				message = string(lines)
			}

			if message == "" && attachFile == "" {
				return fmt.Errorf("--message or --attach is required")
			}

			if dryRun {
				if jsonFlag {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(map[string]any{
						"dryRun":  true,
						"team":    teamName,
						"channel": channelName,
						"message": message,
						"attach":  attachFile,
					})
				}
				fmt.Println("--- Teams Post Preview ---")
				fmt.Printf("Team:     %s\n", teamName)
				fmt.Printf("Channel:  #%s\n", channelName)
				if message != "" {
					fmt.Printf("Message:  %s\n", message)
				}
				if attachFile != "" {
					fmt.Printf("Attach:   %s\n", attachFile)
				}
				fmt.Println("--- Would post via Microsoft Graph API ---")
				return nil
			}

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			tc := graph.NewTeams(client)
			teamID, err := tc.ResolveTeamID(ctx, teamName)
			if err != nil {
				return err
			}
			channelID, err := tc.ResolveChannelID(ctx, teamID, channelName)
			if err != nil {
				return err
			}

			var msg *graph.ChatMessage
			if attachFile != "" {
				msg, err = tc.PostMessageWithFile(ctx, teamID, channelID, message, attachFile)
			} else {
				msg, err = tc.PostMessage(ctx, teamID, channelID, message)
			}
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(msg)
			}

			fmt.Printf("Message posted to #%s\n", channelName)
			if msg.WebURL != "" {
				fmt.Printf("URL: %s\n", msg.WebURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&teamName, "team", "", "Team name or ID (required)")
	cmd.Flags().StringVar(&channelName, "channel", "", "Channel name or ID (required)")
	cmd.Flags().StringVar(&message, "message", "", "Message text")
	cmd.Flags().StringVar(&attachFile, "attach", "", "File to attach")
	cmd.Flags().BoolVar(&useStdin, "stdin", false, "Read message from stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without posting")
	return cmd
}

func newShareCommand() *cobra.Command {
	var (
		teamName    string
		channelName string
		filePath    string
		message     string
		dryRun      bool
	)
	cmd := &cobra.Command{
		Use:   "share",
		Short: "Share a file to a Teams channel with a message",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			if teamName == "" {
				return fmt.Errorf("--team is required")
			}
			if channelName == "" {
				return fmt.Errorf("--channel is required")
			}
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			if dryRun {
				if jsonFlag {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(map[string]any{
						"dryRun":  true,
						"team":    teamName,
						"channel": channelName,
						"file":    filePath,
						"message": message,
					})
				}
				fmt.Println("--- Teams Share Preview ---")
				fmt.Printf("Team:     %s\n", teamName)
				fmt.Printf("Channel:  #%s\n", channelName)
				fmt.Printf("File:     %s\n", filePath)
				if message != "" {
					fmt.Printf("Message:  %s\n", message)
				}
				fmt.Println("--- Would share via Microsoft Graph API ---")
				return nil
			}

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			tc := graph.NewTeams(client)
			teamID, err := tc.ResolveTeamID(ctx, teamName)
			if err != nil {
				return err
			}
			channelID, err := tc.ResolveChannelID(ctx, teamID, channelName)
			if err != nil {
				return err
			}

			msg, err := tc.PostMessageWithFile(ctx, teamID, channelID, message, filePath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(msg)
			}

			fmt.Printf("Shared %s to #%s\n", filePath, channelName)
			return nil
		},
	}
	cmd.Flags().StringVar(&teamName, "team", "", "Team name or ID (required)")
	cmd.Flags().StringVar(&channelName, "channel", "", "Channel name or ID (required)")
	cmd.Flags().StringVar(&filePath, "file", "", "File to share (required)")
	cmd.Flags().StringVar(&message, "message", "", "Accompanying message")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without sharing")
	return cmd
}

func newDMCommand() *cobra.Command {
	var (
		toEmail    string
		message    string
		attachFile string
		dryRun     bool
	)
	cmd := &cobra.Command{
		Use:   "dm",
		Short: "Send a direct message to a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			if toEmail == "" {
				return fmt.Errorf("--to is required")
			}
			if message == "" && attachFile == "" {
				return fmt.Errorf("--message or --attach is required")
			}

			if dryRun {
				if jsonFlag {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(map[string]any{
						"dryRun":  true,
						"to":      toEmail,
						"message": message,
						"attach":  attachFile,
					})
				}
				fmt.Println("--- Teams DM Preview ---")
				fmt.Printf("To:       %s\n", toEmail)
				if message != "" {
					fmt.Printf("Message:  %s\n", message)
				}
				if attachFile != "" {
					fmt.Printf("Attach:   %s\n", attachFile)
				}
				fmt.Println("--- Would send via Microsoft Graph API ---")
				return nil
			}

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			tc := graph.NewTeams(client)

			msgText := message
			if attachFile != "" && msgText == "" {
				msgText = "Shared a file: " + attachFile
			}

			msg, err := tc.SendDirectMessage(ctx, toEmail, msgText)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(msg)
			}

			fmt.Printf("DM sent to %s\n", toEmail)
			return nil
		},
	}
	cmd.Flags().StringVar(&toEmail, "to", "", "Recipient email (required)")
	cmd.Flags().StringVar(&message, "message", "", "Message text")
	cmd.Flags().StringVar(&attachFile, "attach", "", "File to attach")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without sending")
	return cmd
}
