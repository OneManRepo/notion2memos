package memos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client is a Memos API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Memos API client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateMemoRequest represents the request to create a memo
type CreateMemoRequest struct {
	Content     string `json:"content"`
	CreatedTs   int64  `json:"createdTs,omitempty"`
	DisplayTime string `json:"displayTime,omitempty"`
}

// CreateMemoResponse represents the response from creating a memo
type CreateMemoResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	UID       string `json:"uid"`
	Content   string `json:"content"`
	CreatedTs int64  `json:"createdTs"`
}

// CreateMemo creates a new memo in Memos
func (c *Client) CreateMemo(content string, createdTime time.Time, dryRun bool) error {
	if dryRun {
		return c.saveDryRunMemo(content, createdTime)
	}

	// Convert to Unix timestamp (seconds since epoch)
	createdTs := createdTime.Unix()
	// Format as RFC3339 for displayTime field
	displayTime := createdTime.Format(time.RFC3339)
	fmt.Printf("DEBUG: Creating memo with timestamp: %d (%s) displayTime: %s\n", createdTs, createdTime.Format("2006-01-02 15:04:05"), displayTime)

	req := CreateMemoRequest{
		Content:     content,
		CreatedTs:   createdTs,
		DisplayTime: displayTime,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	fmt.Printf("DEBUG: Request body: %s\n", string(body))

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/memos", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var memoResp CreateMemoResponse
	if err := json.NewDecoder(resp.Body).Decode(&memoResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("DEBUG: Response createdTs: %d (%s)\n", memoResp.CreatedTs, time.Unix(memoResp.CreatedTs, 0).Format("2006-01-02 15:04:05"))

	return nil
}

// saveDryRunMemo saves the memo to a file instead of sending it to the API
func (c *Client) saveDryRunMemo(content string, createdTime time.Time) error {
	// Create dry-run-output directory
	outputDir := "./dry-run-output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create dry-run output directory: %w", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("%s.md", createdTime.Format("2006-01-02-150405"))
	filepath := filepath.Join(outputDir, filename)

	// Add metadata header
	fullContent := fmt.Sprintf("---\nCreated: %s\nDry Run: true\n---\n\n%s",
		createdTime.Format("2006-01-02 15:04:05"),
		content)

	// Write to file
	if err := os.WriteFile(filepath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write dry-run file: %w", err)
	}

	return nil
}
