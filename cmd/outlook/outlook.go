// Package outlook provides the "kit outlook" CLI commands for email operations.
package outlook

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/auth"
	"github.com/klytics/m365kit/internal/graph"
)

// NewCommand creates the "outlook" command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "outlook",
		Aliases: []string{"mail"},
		Short:   "Read email, search, and process attachments from Outlook",
		Long:    "Access Microsoft Outlook via Graph API to read emails, download attachments, and process Office files.",
	}

	cmd.AddCommand(newInboxCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newAttachmentsCmd())
	cmd.AddCommand(newDownloadCmd())
	cmd.AddCommand(newMarkReadCmd())
	cmd.AddCommand(newReplyCmd())

	return cmd
}

func newInboxCmd() *cobra.Command {
	var (
		from          string
		subject       string
		hasAttachment bool
		unread        bool
		since         string
		limit         int
	)

	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "List recent emails from your inbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			filter := graph.InboxFilter{
				From:          from,
				Subject:       subject,
				HasAttachment: hasAttachment,
				UnreadOnly:    unread,
				Limit:         limit,
			}

			if since != "" {
				t, err := time.Parse("2006-01-02", since)
				if err != nil {
					return fmt.Errorf("invalid --since date: %w (use YYYY-MM-DD)", err)
				}
				filter.Since = t
			}

			o := graph.NewOutlook(client)
			messages, err := o.ListInbox(cmd.Context(), filter)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(messages)
			}

			if len(messages) == 0 {
				fmt.Println("No messages found.")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, " #\tFROM\tSUBJECT\tRECEIVED\tATTACH\n")
			for i, msg := range messages {
				unreadMark := " "
				if !msg.IsRead {
					unreadMark = "●"
				}
				attach := ""
				if msg.HasAttachments {
					attach = "📎"
				}
				subj := msg.Subject
				if len(subj) > 45 {
					subj = subj[:42] + "..."
				}
				fromAddr := msg.From.EmailAddress.Address
				if len(fromAddr) > 30 {
					fromAddr = fromAddr[:27] + "..."
				}
				fmt.Fprintf(tw, "%s%d\t%s\t%s\t%s\t%s\n",
					unreadMark, i+1, fromAddr, subj,
					graph.FormatEmailDate(msg.ReceivedAt), attach)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Filter by sender email")
	cmd.Flags().StringVar(&subject, "subject", "", "Filter by subject containing text")
	cmd.Flags().BoolVar(&hasAttachment, "has-attachment", false, "Only emails with attachments")
	cmd.Flags().BoolVar(&unread, "unread", false, "Only unread emails")
	cmd.Flags().StringVar(&since, "since", "", "Only emails since date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of emails to return")

	return cmd
}

func newReadCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "read [index]",
		Short: "Read a specific email by index or ID",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			o := graph.NewOutlook(client)
			var msg *graph.EmailMessage

			if id != "" {
				msg, err = o.GetMessage(cmd.Context(), id)
			} else if len(args) == 1 {
				n, parseErr := strconv.Atoi(args[0])
				if parseErr != nil {
					return fmt.Errorf("invalid index: %s", args[0])
				}
				msg, err = o.GetMessageByIndex(cmd.Context(), n)
			} else {
				return fmt.Errorf("provide an index or --id")
			}

			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(msg)
			}

			fmt.Printf("Subject: %s\n", msg.Subject)
			fmt.Printf("From:    %s <%s>\n", msg.From.EmailAddress.Name, msg.From.EmailAddress.Address)
			fmt.Printf("Date:    %s\n", graph.FormatEmailDate(msg.ReceivedAt))
			if msg.HasAttachments {
				fmt.Println("Attach:  Yes")
			}
			fmt.Println()
			fmt.Println(msg.Body.Content)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Message ID (alternative to index)")
	return cmd
}

func newAttachmentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attachments [index]",
		Short: "List attachments on an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			o := graph.NewOutlook(client)
			n, parseErr := strconv.Atoi(args[0])
			if parseErr != nil {
				return fmt.Errorf("invalid index: %s", args[0])
			}

			msg, err := o.GetMessageByIndex(cmd.Context(), n)
			if err != nil {
				return err
			}

			attachments, err := o.ListAttachments(cmd.Context(), msg.ID)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(attachments)
			}

			if len(attachments) == 0 {
				fmt.Println("No attachments.")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "NAME\tSIZE\tTYPE\n")
			for _, att := range attachments {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", att.Name, formatSize(att.Size), att.ContentType)
			}
			tw.Flush()
			return nil
		},
	}
}

func newDownloadCmd() *cobra.Command {
	var (
		output     string
		officeOnly bool
	)

	cmd := &cobra.Command{
		Use:   "download [index]",
		Short: "Download attachments from an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			o := graph.NewOutlook(client)
			n, parseErr := strconv.Atoi(args[0])
			if parseErr != nil {
				return fmt.Errorf("invalid index: %s", args[0])
			}

			msg, err := o.GetMessageByIndex(cmd.Context(), n)
			if err != nil {
				return err
			}

			attachments, err := o.ListAttachments(cmd.Context(), msg.ID)
			if err != nil {
				return err
			}

			if output == "" {
				output = "."
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			var downloaded []string

			for _, att := range attachments {
				if att.IsInline {
					continue
				}
				if officeOnly && !graph.IsOfficeAttachment(att.Name) {
					continue
				}

				path, err := o.DownloadAttachment(cmd.Context(), msg.ID, att.ID, output)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not download %s: %v\n", att.Name, err)
					continue
				}
				downloaded = append(downloaded, path)
				if !jsonOut {
					fmt.Printf("Downloaded: %s\n", path)
				}
			}

			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]any{
					"downloaded": downloaded,
					"count":      len(downloaded),
				})
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory (default: current)")
	cmd.Flags().BoolVar(&officeOnly, "office-only", false, "Only download Office files (.docx/.xlsx/.pptx/.pdf)")

	return cmd
}

func newMarkReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mark-read [index]",
		Short: "Mark an email as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			o := graph.NewOutlook(client)
			n, parseErr := strconv.Atoi(args[0])
			if parseErr != nil {
				return fmt.Errorf("invalid index: %s", args[0])
			}

			msg, err := o.GetMessageByIndex(cmd.Context(), n)
			if err != nil {
				return err
			}

			if err := o.MarkAsRead(cmd.Context(), msg.ID); err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"marked": msg.ID})
			}

			fmt.Printf("Marked as read: %s\n", msg.Subject)
			return nil
		},
	}
}

func newReplyCmd() *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:   "reply [index]",
		Short: "Reply to an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if body == "" {
				return fmt.Errorf("--body is required")
			}

			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			o := graph.NewOutlook(client)
			n, parseErr := strconv.Atoi(args[0])
			if parseErr != nil {
				return fmt.Errorf("invalid index: %s", args[0])
			}

			msg, err := o.GetMessageByIndex(cmd.Context(), n)
			if err != nil {
				return err
			}

			if err := o.Reply(cmd.Context(), msg.ID, body); err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"replied": msg.ID})
			}

			fmt.Printf("Replied to: %s\n", msg.Subject)
			return nil
		},
	}

	cmd.Flags().StringVar(&body, "body", "", "Reply body text")
	return cmd
}

func formatSize(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", bytes, units[0])
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", size), "0"), ".") + " " + units[i]
}
