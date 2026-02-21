// Package onedrive provides CLI commands for OneDrive file operations.
package onedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/auth"
	"github.com/klytics/m365kit/internal/graph"
)

// NewCommand returns the onedrive command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onedrive",
		Short: "Manage OneDrive files",
		Long:  "List, upload, download, search, and share files on Microsoft OneDrive.",
	}

	cmd.AddCommand(newLsCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newPutCommand())
	cmd.AddCommand(newRecentCommand())
	cmd.AddCommand(newSearchCommand())
	cmd.AddCommand(newShareCommand())

	return cmd
}

func newLsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [path]",
		Short: "List files in a OneDrive folder",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			folderPath := "/"
			if len(args) > 0 {
				folderPath = args[0]
			}

			od := graph.NewOneDrive(client)
			items, err := od.ListFolder(ctx, folderPath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}

			if len(items) == 0 {
				fmt.Println("(empty folder)")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TYPE\tNAME\tSIZE\tMODIFIED\n")
			for _, item := range items {
				itemType := "file"
				if item.IsFolder {
					itemType = "dir"
				}
				size := graph.FormatSize(item.Size)
				if item.IsFolder {
					size = fmt.Sprintf("%d items", item.ChildCount)
				}
				modified := item.LastModifiedAt.Format("2006-01-02 15:04")
				name := item.Name
				if item.IsFolder {
					name = color.New(color.FgBlue, color.Bold).Sprint(name + "/")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", itemType, name, size, modified)
			}
			return w.Flush()
		},
	}
}

func newGetCommand() *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "get <remote-path>",
		Short: "Download a file from OneDrive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			remotePath := args[0]
			if outputPath == "" {
				outputPath = filepath.Base(remotePath)
			}

			od := graph.NewOneDrive(client)
			n, err := od.DownloadFile(ctx, remotePath, outputPath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"remote": remotePath,
					"local":  outputPath,
					"bytes":  n,
				})
			}

			fmt.Printf("Downloaded %s → %s (%s)\n", remotePath, outputPath, graph.FormatSize(n))
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Local output path (default: filename)")
	return cmd
}

func newPutCommand() *cobra.Command {
	var remotePath string
	cmd := &cobra.Command{
		Use:   "put <local-file>",
		Short: "Upload a file to OneDrive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			localPath := args[0]
			if remotePath == "" {
				remotePath = filepath.Base(localPath)
			}

			od := graph.NewOneDrive(client)
			item, err := od.UploadFile(ctx, localPath, remotePath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"local":  localPath,
					"remote": remotePath,
					"id":     item.ID,
					"webUrl": item.WebURL,
					"size":   item.Size,
				})
			}

			fmt.Printf("Uploaded %s → %s (%s)\n", localPath, remotePath, graph.FormatSize(item.Size))
			if item.WebURL != "" {
				fmt.Printf("Web: %s\n", item.WebURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&remotePath, "remote", "r", "", "Remote path (default: filename)")
	return cmd
}

func newRecentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "recent",
		Short: "List recently accessed files",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			od := graph.NewOneDrive(client)
			items, err := od.RecentFiles(ctx)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}

			if len(items) == 0 {
				fmt.Println("No recent files")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tSIZE\tMODIFIED\n")
			for _, item := range items {
				size := graph.FormatSize(item.Size)
				modified := item.LastModifiedAt.Format("2006-01-02 15:04")
				fmt.Fprintf(w, "%s\t%s\t%s\n", item.Name, size, modified)
			}
			return w.Flush()
		},
	}
}

func newSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search OneDrive files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			od := graph.NewOneDrive(client)
			items, err := od.SearchFiles(ctx, args[0])
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}

			if len(items) == 0 {
				fmt.Printf("No files matching %q\n", args[0])
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tSIZE\tPATH\n")
			for _, item := range items {
				size := graph.FormatSize(item.Size)
				path := item.ParentPath
				if idx := strings.Index(path, ":"); idx >= 0 {
					path = path[idx+1:]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", item.Name, size, path)
			}
			return w.Flush()
		},
	}
}

func newShareCommand() *cobra.Command {
	var linkType string
	cmd := &cobra.Command{
		Use:   "share <remote-path>",
		Short: "Create a sharing link for a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			od := graph.NewOneDrive(client)
			link, err := od.CreateShareLink(ctx, args[0], linkType)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"path": args[0],
					"type": linkType,
					"url":  link,
				})
			}

			fmt.Printf("Share link (%s): %s\n", linkType, link)
			return nil
		},
	}
	cmd.Flags().StringVar(&linkType, "type", "view", "Link type: view | edit")
	return cmd
}
