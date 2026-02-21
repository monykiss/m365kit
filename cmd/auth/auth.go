// Package auth provides CLI commands for Microsoft 365 authentication.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/auth"
)

// NewCommand returns the auth command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Microsoft 365",
		Long: `Manage Microsoft 365 authentication for OneDrive and SharePoint access.

Setup:
  1. Register an Azure AD app at portal.azure.com
  2. Set: export KIT_AZURE_CLIENT_ID="your-app-client-id"
  3. Run: kit auth login`,
	}

	cmd.AddCommand(newLoginCommand())
	cmd.AddCommand(newWhoAmICommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newLogoutCommand())
	cmd.AddCommand(newRefreshCommand())

	return cmd
}

func newLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Microsoft 365 (device code flow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID := os.Getenv("KIT_AZURE_CLIENT_ID")
			if clientID == "" {
				return fmt.Errorf("KIT_AZURE_CLIENT_ID is not set\n\nSetup:\n  1. Register an Azure AD app at portal.azure.com\n  2. export KIT_AZURE_CLIENT_ID=\"your-app-client-id\"\n  3. kit auth login")
			}

			ctx := context.Background()
			token, err := auth.DeviceCodeFlow(ctx, clientID)
			if err != nil {
				return err
			}

			// Fetch user info
			client := &http.Client{
				Transport: &auth.BearerTransport{Token: token.AccessToken},
			}
			name, email, err := auth.WhoAmI(ctx, client)
			if err != nil {
				fmt.Println("Authenticated (could not fetch user details)")
				return nil
			}

			green := color.New(color.FgGreen)
			green.Printf("Authenticated as %s (%s)\n", name, email)
			fmt.Println("Token saved to ~/.kit/token.json")
			return nil
		},
	}
}

func newWhoAmICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			ctx := context.Background()

			client, err := auth.RequireAuth(ctx)
			if err != nil {
				return err
			}

			token, _ := auth.LoadToken()

			name, email, err := auth.WhoAmI(ctx, client)
			if err != nil {
				return fmt.Errorf("could not fetch user info: %w", err)
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"name":      name,
					"email":     email,
					"expiresIn": int(token.ExpiresIn().Minutes()),
				})
			}

			fmt.Printf("%s (%s)\n", name, email)
			if token != nil {
				fmt.Printf("Token expires in %d minutes\n", int(token.ExpiresIn().Minutes()))
			}
			return nil
		},
	}
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			token, err := auth.LoadToken()
			if err != nil {
				if jsonFlag {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(map[string]any{
						"authenticated": false,
						"error":         err.Error(),
					})
				}
				fmt.Println("Not authenticated — run: kit auth login")
				return nil
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"authenticated": true,
					"expired":       token.IsExpired(),
					"expiresAt":     token.ExpiresAt.Format(time.RFC3339),
					"expiresIn":     int(token.ExpiresIn().Minutes()),
					"scopes":        auth.Scopes(),
				})
			}

			if token.IsExpired() {
				color.New(color.FgRed).Println("Token expired — run: kit auth login")
				return nil
			}

			green := color.New(color.FgGreen)
			green.Print("Authenticated")

			// Try to get user info
			ctx := context.Background()
			client, authErr := auth.RequireAuth(ctx)
			if authErr == nil {
				if name, email, err := auth.WhoAmI(ctx, client); err == nil {
					green.Printf(": %s (%s)", name, email)
				}
			}
			fmt.Println()

			fmt.Printf("Token expires: %s (%d minutes)\n",
				token.ExpiresAt.Format("2006-01-02 15:04"),
				int(token.ExpiresIn().Minutes()))

			scopes := auth.Scopes()
			filtered := make([]string, 0, len(scopes))
			for _, s := range scopes {
				if s != "offline_access" {
					filtered = append(filtered, s)
				}
			}
			fmt.Printf("Scopes: %s\n", strings.Join(filtered, ", "))
			return nil
		},
	}
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke authentication (delete token)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.DeleteToken(); err != nil {
				return err
			}
			fmt.Println("Logged out — token deleted")
			return nil
		},
	}
}

func newRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the authentication token",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := auth.LoadToken()
			if err != nil {
				return err
			}

			clientID := os.Getenv("KIT_AZURE_CLIENT_ID")
			if clientID == "" {
				return fmt.Errorf("KIT_AZURE_CLIENT_ID not set — see: kit auth --help")
			}

			ctx := context.Background()
			newToken, err := auth.RefreshIfNeeded(ctx, token, clientID)
			if err != nil {
				return err
			}

			if newToken.AccessToken == token.AccessToken {
				fmt.Printf("Token still valid (%d minutes remaining)\n", int(token.ExpiresIn().Minutes()))
			} else {
				fmt.Printf("Token refreshed — expires in %d minutes\n", int(newToken.ExpiresIn().Minutes()))
			}
			return nil
		},
	}
}
