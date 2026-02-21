// Package sharepoint provides CLI commands for SharePoint document management.
package sharepoint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/auth"
	"github.com/klytics/m365kit/internal/graph"
)

// NewCommand returns the sharepoint command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sharepoint",
		Aliases: []string{"sp"},
		Short:   "Manage SharePoint sites and document libraries",
		Long:    "List sites, browse document libraries, and manage files on SharePoint.",
	}

	cmd.AddCommand(newSitesCommand())
	cmd.AddCommand(newLibsCommand())
	cmd.AddCommand(newLsCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newPutCommand())
	cmd.AddCommand(newAuditCommand())

	return cmd
}

func newSitesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sites [search-query]",
		Short: "List SharePoint sites",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			sp := graph.NewSharePoint(client)
			sites, err := sp.ListSites(ctx, query)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(sites)
			}

			if len(sites) == 0 {
				fmt.Println("No SharePoint sites found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tURL\tDESCRIPTION\n")
			for _, s := range sites {
				desc := s.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", s.DisplayName, s.WebURL, desc)
			}
			return w.Flush()
		},
	}
}

func newLibsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "libs <site-id>",
		Short: "List document libraries for a SharePoint site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			sp := graph.NewSharePoint(client)
			libs, err := sp.ListLibraries(ctx, args[0])
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(libs)
			}

			if len(libs) == 0 {
				fmt.Println("No document libraries found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tNAME\tTYPE\tURL\n")
			for _, lib := range libs {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", lib.ID, lib.DisplayName, lib.DriveType, lib.WebURL)
			}
			return w.Flush()
		},
	}
}

func newLsCommand() *cobra.Command {
	var driveID string
	cmd := &cobra.Command{
		Use:   "ls <site-id> [path]",
		Short: "List files in a SharePoint document library",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			siteID := args[0]
			folderPath := "/"
			if len(args) > 1 {
				folderPath = args[1]
			}

			sp := graph.NewSharePoint(client)

			// If no drive ID specified, use the first library
			if driveID == "" {
				libs, err := sp.ListLibraries(ctx, siteID)
				if err != nil {
					return fmt.Errorf("could not list libraries: %w", err)
				}
				if len(libs) == 0 {
					return fmt.Errorf("no document libraries found on this site")
				}
				driveID = libs[0].ID
			}

			items, err := sp.ListLibraryFiles(ctx, siteID, driveID, folderPath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}

			if len(items) == 0 {
				fmt.Println("(empty)")
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
	cmd.Flags().StringVar(&driveID, "drive", "", "Document library (drive) ID (default: first library)")
	return cmd
}

func newGetCommand() *cobra.Command {
	var driveID, outputPath string
	cmd := &cobra.Command{
		Use:   "get <site-id> <remote-path>",
		Short: "Download a file from a SharePoint library",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			siteID := args[0]
			remotePath := args[1]
			if outputPath == "" {
				outputPath = filepath.Base(remotePath)
			}

			sp := graph.NewSharePoint(client)

			if driveID == "" {
				libs, err := sp.ListLibraries(ctx, siteID)
				if err != nil {
					return err
				}
				if len(libs) == 0 {
					return fmt.Errorf("no document libraries found")
				}
				driveID = libs[0].ID
			}

			n, err := sp.DownloadFromLibrary(ctx, siteID, driveID, remotePath, outputPath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"site":   siteID,
					"remote": remotePath,
					"local":  outputPath,
					"bytes":  n,
				})
			}

			fmt.Printf("Downloaded %s → %s (%s)\n", remotePath, outputPath, graph.FormatSize(n))
			return nil
		},
	}
	cmd.Flags().StringVar(&driveID, "drive", "", "Document library (drive) ID")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Local output path")
	return cmd
}

func newPutCommand() *cobra.Command {
	var driveID, remotePath string
	cmd := &cobra.Command{
		Use:   "put <site-id> <local-file>",
		Short: "Upload a file to a SharePoint library",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			siteID := args[0]
			localPath := args[1]
			if remotePath == "" {
				remotePath = filepath.Base(localPath)
			}

			sp := graph.NewSharePoint(client)

			if driveID == "" {
				libs, err := sp.ListLibraries(ctx, siteID)
				if err != nil {
					return err
				}
				if len(libs) == 0 {
					return fmt.Errorf("no document libraries found")
				}
				driveID = libs[0].ID
			}

			item, err := sp.UploadToLibrary(ctx, siteID, driveID, remotePath, localPath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"site":   siteID,
					"local":  localPath,
					"remote": remotePath,
					"id":     item.ID,
					"webUrl": item.WebURL,
				})
			}

			fmt.Printf("Uploaded %s → %s\n", localPath, remotePath)
			if item.WebURL != "" {
				fmt.Printf("Web: %s\n", item.WebURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&driveID, "drive", "", "Document library (drive) ID")
	cmd.Flags().StringVarP(&remotePath, "remote", "r", "", "Remote path")
	return cmd
}

func newAuditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "audit <site-id>",
		Short: "Show recent activity on a SharePoint site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			sp := graph.NewSharePoint(client)
			entries, err := sp.AuditSite(ctx, args[0])
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			if len(entries) == 0 {
				fmt.Println("No recent activity")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TIME\tACTION\tUSER\tFILE\n")
			for _, e := range entries {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					e.OccurredAt.Format("2006-01-02 15:04"),
					e.Action, e.Actor, e.ItemName)
			}
			return w.Flush()
		},
	}
}
