package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Site represents a SharePoint site.
type Site struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Name        string    `json:"name"`
	WebURL      string    `json:"webUrl"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdDateTime"`
}

// DocumentLibrary represents a SharePoint document library.
type DocumentLibrary struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Name        string    `json:"name"`
	WebURL      string    `json:"webUrl"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdDateTime"`
	DriveType   string    `json:"driveType"`
}

// AuditEntry represents an activity log entry.
type AuditEntry struct {
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	ItemName   string    `json:"itemName"`
	ItemPath   string    `json:"itemPath"`
	OccurredAt time.Time `json:"occurredAt"`
}

// SharePoint provides operations on Microsoft SharePoint.
type SharePoint struct {
	Client *http.Client
}

// NewSharePoint creates a new SharePoint client with an authenticated HTTP client.
func NewSharePoint(client *http.Client) *SharePoint {
	return &SharePoint{Client: client}
}

type sitesResponse struct {
	Value []Site `json:"value"`
}

type drivesResponse struct {
	Value []DocumentLibrary `json:"value"`
}

// ListSites returns SharePoint sites the user has access to.
func (sp *SharePoint) ListSites(ctx context.Context, query string) ([]Site, error) {
	var endpoint string
	if query != "" {
		endpoint = graphBase + "/sites?search=" + url.QueryEscape(query)
	} else {
		endpoint = graphBase + "/sites?search=*"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SharePoint sites request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SharePoint API returned %d: %s", resp.StatusCode, string(body))
	}

	var result sitesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse sites response: %w", err)
	}

	return result.Value, nil
}

// GetSite returns a specific site by hostname and path.
// siteRef can be "hostname:/path" or a site ID.
func (sp *SharePoint) GetSite(ctx context.Context, siteRef string) (*Site, error) {
	var endpoint string
	if strings.Contains(siteRef, ":") || strings.Contains(siteRef, ".") {
		endpoint = graphBase + "/sites/" + siteRef
	} else {
		endpoint = graphBase + "/sites/" + siteRef
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SharePoint get site failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SharePoint API returned %d: %s", resp.StatusCode, string(body))
	}

	var site Site
	if err := json.Unmarshal(body, &site); err != nil {
		return nil, fmt.Errorf("could not parse site: %w", err)
	}

	return &site, nil
}

// ListLibraries returns document libraries for a site.
func (sp *SharePoint) ListLibraries(ctx context.Context, siteID string) ([]DocumentLibrary, error) {
	endpoint := graphBase + "/sites/" + siteID + "/drives"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SharePoint libraries request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SharePoint API returned %d: %s", resp.StatusCode, string(body))
	}

	var result drivesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse libraries response: %w", err)
	}

	return result.Value, nil
}

// ListLibraryFiles lists files in a specific document library.
func (sp *SharePoint) ListLibraryFiles(ctx context.Context, siteID, driveID, folderPath string) ([]DriveItem, error) {
	var endpoint string
	folderPath = strings.TrimRight(folderPath, "/")
	if folderPath == "" || folderPath == "/" {
		endpoint = graphBase + "/sites/" + siteID + "/drives/" + driveID + "/root/children"
	} else {
		endpoint = graphBase + "/sites/" + siteID + "/drives/" + driveID + "/root:/" + url.PathEscape(folderPath) + ":/children"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SharePoint list files request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SharePoint API returned %d: %s", resp.StatusCode, string(body))
	}

	var result driveItemsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse library files: %w", err)
	}

	return result.Value, nil
}

// DownloadFromLibrary downloads a file from a SharePoint document library.
func (sp *SharePoint) DownloadFromLibrary(ctx context.Context, siteID, driveID, itemPath, localPath string) (int64, error) {
	endpoint := graphBase + "/sites/" + siteID + "/drives/" + driveID + "/root:/" + url.PathEscape(itemPath) + ":/content"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, err
	}

	resp, err := sp.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("SharePoint download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("download failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	f, err := createLocalFile(localPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return io.Copy(f, resp.Body)
}

// UploadToLibrary uploads a file to a SharePoint document library (up to 4MB).
func (sp *SharePoint) UploadToLibrary(ctx context.Context, siteID, driveID, remotePath, localPath string) (*DriveItem, error) {
	f, err := openAndValidateFile(localPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	endpoint := graphBase + "/sites/" + siteID + "/drives/" + driveID + "/root:/" + url.PathEscape(remotePath) + ":/content"
	req, err := http.NewRequestWithContext(ctx, "PUT", endpoint, f)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := sp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("upload failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var item DriveItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("could not parse upload response: %w", err)
	}

	return &item, nil
}

// AuditSite returns recent activity for a site's primary drive.
func (sp *SharePoint) AuditSite(ctx context.Context, siteID string) ([]AuditEntry, error) {
	// First get the default drive
	libs, err := sp.ListLibraries(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("could not list libraries for audit: %w", err)
	}
	if len(libs) == 0 {
		return nil, fmt.Errorf("no document libraries found on site")
	}

	driveID := libs[0].ID
	endpoint := graphBase + "/sites/" + siteID + "/drives/" + driveID + "/activities"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("audit request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// Activities API may not be available — return helpful message
		return nil, fmt.Errorf("activities API returned %d — this requires SharePoint admin permissions", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			Action json.RawMessage `json:"action"`
			Actor  struct {
				User struct {
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"actor"`
			Times struct {
				Recorded string `json:"recordedDateTime"`
			} `json:"times"`
			DriveItem struct {
				Name string `json:"name"`
			} `json:"driveItem"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse audit response: %w", err)
	}

	var entries []AuditEntry
	for _, v := range result.Value {
		action := "unknown"
		// Parse action type from JSON keys
		var actionMap map[string]any
		if err := json.Unmarshal(v.Action, &actionMap); err == nil {
			for k := range actionMap {
				action = k
				break
			}
		}

		recorded, _ := time.Parse(time.RFC3339, v.Times.Recorded)
		entries = append(entries, AuditEntry{
			Action:     action,
			Actor:      v.Actor.User.DisplayName,
			ItemName:   v.DriveItem.Name,
			OccurredAt: recorded,
		})
	}

	return entries, nil
}

// helper: open file and validate size for upload
func openAndValidateFile(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("could not stat file: %w", err)
	}
	if info.Size() > 4*1024*1024 {
		f.Close()
		return nil, fmt.Errorf("file too large (%d bytes, max 4MB)", info.Size())
	}
	return f, nil
}

// helper: create local file for download
func createLocalFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("could not create directory: %w", err)
	}
	return os.Create(path)
}
