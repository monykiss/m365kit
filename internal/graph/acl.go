package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Permission represents a SharePoint permission entry.
type Permission struct {
	ID            string     `json:"id"`
	Roles         []string   `json:"roles"`
	GrantedTo     *Principal `json:"grantedTo,omitempty"`
	GrantedToV2   *Principal `json:"grantedToV2,omitempty"`
	Link          *PermLink  `json:"link,omitempty"`
	InheritedFrom *struct {
		ID string `json:"id"`
	} `json:"inheritedFrom,omitempty"`
}

// IsInherited returns true if this permission is inherited from a parent.
func (p Permission) IsInherited() bool {
	return p.InheritedFrom != nil
}

// IsExternal checks if this permission grants access to an external user.
func (p Permission) IsExternal(orgDomain string) bool {
	email := p.GetEmail()
	if email == "" {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	return !strings.EqualFold(parts[1], orgDomain)
}

// GetEmail returns the email address of the granted principal.
func (p Permission) GetEmail() string {
	if p.GrantedToV2 != nil && p.GrantedToV2.User != nil {
		return p.GrantedToV2.User.Email
	}
	if p.GrantedTo != nil && p.GrantedTo.User != nil {
		return p.GrantedTo.User.Email
	}
	return ""
}

// IsAnonymousLink returns true if this is an anonymous sharing link.
func (p Permission) IsAnonymousLink() bool {
	return p.Link != nil && p.Link.Scope == "anonymous"
}

// Principal represents a user or group with access.
type Principal struct {
	User  *GraphUser  `json:"user,omitempty"`
	Group *GraphGroup `json:"group,omitempty"`
}

// GraphUser represents a user in the Graph API.
type GraphUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// GraphGroup represents a group in the Graph API.
type GraphGroup struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// PermLink represents a sharing link.
type PermLink struct {
	Scope string `json:"scope"` // "anonymous", "organization", "users"
	Type  string `json:"type"`  // "view", "edit"
	URL   string `json:"webUrl"`
}

// ACLReport holds the complete result of a permissions audit.
type ACLReport struct {
	Site              string     `json:"site"`
	GeneratedAt       time.Time  `json:"generatedAt"`
	TotalFiles        int        `json:"totalFiles"`
	ExternalShares    int        `json:"externalShares"`
	BrokenInheritance int        `json:"brokenInheritance"`
	AnonymousLinks    int        `json:"anonymousLinks"`
	Entries           []ACLEntry `json:"entries"`
}

// ACLEntry represents the permissions on a single file.
type ACLEntry struct {
	Path                 string       `json:"path"`
	Permissions          []Permission `json:"permissions"`
	HasUniquePermissions bool         `json:"hasUniquePermissions"`
	ExternalUsers        []string     `json:"externalUsers,omitempty"`
}

// ACL provides SharePoint permissions audit operations.
type ACL struct {
	Client    *http.Client
	OrgDomain string // e.g., "company.com" — used for external detection
}

// NewACL creates a new ACL client.
func NewACL(client *http.Client, orgDomain string) *ACL {
	return &ACL{Client: client, OrgDomain: orgDomain}
}

type permissionsResponse struct {
	Value []Permission `json:"value"`
}

// GetFilePermissions returns permissions for a specific file.
func (a *ACL) GetFilePermissions(ctx context.Context, siteID, driveID, itemID string) ([]Permission, error) {
	endpoint := graphBase + "/sites/" + siteID + "/drives/" + driveID + "/items/" + url.PathEscape(itemID) + "/permissions"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get permissions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get permissions failed (%d): %s", resp.StatusCode, string(body))
	}

	var result permissionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("could not parse permissions: %w", err)
	}
	return result.Value, nil
}

// AuditSitePermissions scans files in a site's default drive and returns an ACL report.
func (a *ACL) AuditSitePermissions(ctx context.Context, siteID string) (*ACLReport, error) {
	// Get the default drive
	endpoint := graphBase + "/sites/" + siteID + "/drive"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get site drive: %w", err)
	}
	defer resp.Body.Close()

	var drive struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&drive)

	if drive.ID == "" {
		return nil, fmt.Errorf("could not determine default drive for site")
	}

	return a.AuditDrive(ctx, siteID, drive.ID)
}

// AuditDrive scans all files in a drive and returns their permissions.
func (a *ACL) AuditDrive(ctx context.Context, siteID, driveID string) (*ACLReport, error) {
	// List all items in the drive
	endpoint := graphBase + "/sites/" + siteID + "/drives/" + driveID + "/root/children"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not list drive items: %w", err)
	}
	defer resp.Body.Close()

	var itemsResp struct {
		Value []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"value"`
	}
	json.NewDecoder(resp.Body).Decode(&itemsResp)

	report := &ACLReport{
		Site:        siteID,
		GeneratedAt: time.Now(),
	}

	for _, item := range itemsResp.Value {
		perms, err := a.GetFilePermissions(ctx, siteID, driveID, item.ID)
		if err != nil {
			continue
		}

		entry := ACLEntry{
			Path:        item.Name,
			Permissions: perms,
		}

		// Analyze permissions
		for _, p := range perms {
			if !p.IsInherited() {
				entry.HasUniquePermissions = true
			}
			if p.IsExternal(a.OrgDomain) {
				entry.ExternalUsers = append(entry.ExternalUsers, p.GetEmail())
			}
			if p.IsAnonymousLink() {
				report.AnonymousLinks++
			}
		}

		if entry.HasUniquePermissions {
			report.BrokenInheritance++
		}
		if len(entry.ExternalUsers) > 0 {
			report.ExternalShares++
		}

		report.Entries = append(report.Entries, entry)
		report.TotalFiles++
	}

	return report, nil
}

// FindExternalShares returns entries with external user access.
func FindExternalShares(report *ACLReport) []ACLEntry {
	var result []ACLEntry
	for _, entry := range report.Entries {
		if len(entry.ExternalUsers) > 0 {
			result = append(result, entry)
		}
	}
	return result
}

// FindBrokenInheritance returns entries with unique (non-inherited) permissions.
func FindBrokenInheritance(report *ACLReport) []ACLEntry {
	var result []ACLEntry
	for _, entry := range report.Entries {
		if entry.HasUniquePermissions {
			result = append(result, entry)
		}
	}
	return result
}

// CountAnonymousLinks returns the total anonymous link count.
func CountAnonymousLinks(report *ACLReport) int {
	return report.AnonymousLinks
}
