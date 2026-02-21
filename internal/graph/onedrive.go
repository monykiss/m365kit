// Package graph provides Microsoft Graph API clients for OneDrive and SharePoint.
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

const graphBase = "https://graph.microsoft.com/v1.0"

// DriveItem represents a file or folder in OneDrive.
type DriveItem struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Size             int64     `json:"size"`
	WebURL           string    `json:"webUrl"`
	LastModifiedAt   time.Time `json:"lastModifiedDateTime"`
	CreatedAt        time.Time `json:"createdDateTime"`
	IsFolder         bool      `json:"-"`
	ChildCount       int       `json:"-"`
	MimeType         string    `json:"-"`
	DownloadURL      string    `json:"-"`
	ParentPath       string    `json:"-"`
	SharingLink      string    `json:"-"`
}

// UnmarshalJSON implements custom unmarshalling for DriveItem.
func (d *DriveItem) UnmarshalJSON(data []byte) error {
	type Alias DriveItem
	aux := &struct {
		*Alias
		Folder *struct {
			ChildCount int `json:"childCount"`
		} `json:"folder"`
		File *struct {
			MimeType string `json:"mimeType"`
		} `json:"file"`
		DownloadURL      string `json:"@microsoft.graph.downloadUrl"`
		ParentReference  *struct {
			Path string `json:"path"`
		} `json:"parentReference"`
		LastModified string `json:"lastModifiedDateTime"`
		Created      string `json:"createdDateTime"`
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.Folder != nil {
		d.IsFolder = true
		d.ChildCount = aux.Folder.ChildCount
	}
	if aux.File != nil {
		d.MimeType = aux.File.MimeType
	}
	d.DownloadURL = aux.DownloadURL
	if aux.ParentReference != nil {
		d.ParentPath = aux.ParentReference.Path
	}
	if aux.LastModified != "" {
		if t, err := time.Parse(time.RFC3339, aux.LastModified); err == nil {
			d.LastModifiedAt = t
		}
	}
	if aux.Created != "" {
		if t, err := time.Parse(time.RFC3339, aux.Created); err == nil {
			d.CreatedAt = t
		}
	}

	return nil
}

type driveItemsResponse struct {
	Value    []DriveItem `json:"value"`
	NextLink string      `json:"@odata.nextLink"`
}

// OneDrive provides operations on Microsoft OneDrive.
type OneDrive struct {
	Client *http.Client
}

// NewOneDrive creates a new OneDrive client with an authenticated HTTP client.
func NewOneDrive(client *http.Client) *OneDrive {
	return &OneDrive{Client: client}
}

// ListFolder lists items in a OneDrive folder by path.
// Use "/" or "" for root.
func (o *OneDrive) ListFolder(ctx context.Context, folderPath string) ([]DriveItem, error) {
	var endpoint string
	folderPath = strings.TrimRight(folderPath, "/")
	if folderPath == "" || folderPath == "/" {
		endpoint = graphBase + "/me/drive/root/children"
	} else {
		endpoint = graphBase + "/me/drive/root:/" + url.PathEscape(folderPath) + ":/children"
	}

	var allItems []DriveItem
	for endpoint != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		resp, err := o.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("OneDrive list request failed: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("OneDrive API returned %d: %s", resp.StatusCode, string(body))
		}

		var result driveItemsResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("could not parse OneDrive response: %w", err)
		}

		allItems = append(allItems, result.Value...)
		endpoint = result.NextLink
	}

	return allItems, nil
}

// GetItem returns metadata for a single item by path.
func (o *OneDrive) GetItem(ctx context.Context, itemPath string) (*DriveItem, error) {
	itemPath = strings.TrimRight(itemPath, "/")
	endpoint := graphBase + "/me/drive/root:/" + url.PathEscape(itemPath)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OneDrive get request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OneDrive API returned %d: %s", resp.StatusCode, string(body))
	}

	var item DriveItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("could not parse item: %w", err)
	}

	return &item, nil
}

// DownloadFile downloads a file from OneDrive to a local path.
func (o *OneDrive) DownloadFile(ctx context.Context, remotePath, localPath string) (int64, error) {
	item, err := o.GetItem(ctx, remotePath)
	if err != nil {
		return 0, err
	}

	if item.DownloadURL == "" {
		return 0, fmt.Errorf("no download URL available for %s", remotePath)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", item.DownloadURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}

	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("could not create directory: %w", err)
	}

	f, err := os.Create(localPath)
	if err != nil {
		return 0, fmt.Errorf("could not create local file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("download copy failed: %w", err)
	}

	return n, nil
}

// UploadFile uploads a local file to OneDrive (files up to 4MB).
func (o *OneDrive) UploadFile(ctx context.Context, localPath, remotePath string) (*DriveItem, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("could not open local file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat local file: %w", err)
	}

	if info.Size() > 4*1024*1024 {
		return nil, fmt.Errorf("file too large for simple upload (%d bytes, max 4MB) — use OneDrive web for large files", info.Size())
	}

	endpoint := graphBase + "/me/drive/root:/" + url.PathEscape(remotePath) + ":/content"
	req, err := http.NewRequestWithContext(ctx, "PUT", endpoint, f)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := o.Client.Do(req)
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

// RecentFiles returns recently accessed files.
func (o *OneDrive) RecentFiles(ctx context.Context) ([]DriveItem, error) {
	endpoint := graphBase + "/me/drive/recent"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("recent files request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OneDrive API returned %d: %s", resp.StatusCode, string(body))
	}

	var result driveItemsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse recent files: %w", err)
	}

	return result.Value, nil
}

// SearchFiles searches for files in OneDrive by query string.
func (o *OneDrive) SearchFiles(ctx context.Context, query string) ([]DriveItem, error) {
	endpoint := graphBase + "/me/drive/root/search(q='" + url.QueryEscape(query) + "')"
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OneDrive API returned %d: %s", resp.StatusCode, string(body))
	}

	var result driveItemsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse search results: %w", err)
	}

	return result.Value, nil
}

// CreateShareLink creates a sharing link for a file.
func (o *OneDrive) CreateShareLink(ctx context.Context, itemPath, linkType string) (string, error) {
	if linkType == "" {
		linkType = "view"
	}

	item, err := o.GetItem(ctx, itemPath)
	if err != nil {
		return "", err
	}

	endpoint := graphBase + "/me/drive/items/" + item.ID + "/createLink"
	payload := fmt.Sprintf(`{"type":"%s","scope":"anonymous"}`, linkType)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("share link request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("share link failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Link struct {
			WebURL string `json:"webUrl"`
		} `json:"link"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("could not parse share link response: %w", err)
	}

	return result.Link.WebURL, nil
}

// FormatSize returns a human-readable file size string.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
