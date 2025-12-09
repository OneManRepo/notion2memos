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
	Content string `json:"content"`
}

// UpdateMemoRequest represents the request to update memo fields
type UpdateMemoRequest struct {
	DisplayTime string `json:"displayTime,omitempty"`
}

// CreateMemoResponse represents the response from creating a memo
type CreateMemoResponse struct {
	Name        string `json:"name"`
	Creator     string `json:"creator"`
	CreateTime  string `json:"createTime"`
	UpdateTime  string `json:"updateTime"`
	DisplayTime string `json:"displayTime"`
	Content     string `json:"content"`
}

// CreateMemo creates a new memo in Memos
func (c *Client) CreateMemo(content string, createdTime time.Time, dryRun bool) error {
	if dryRun {
		return c.saveDryRunMemo(content, createdTime)
	}

	// Step 1: Create the memo
	req := CreateMemoRequest{
		Content: content,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

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

	// Parse the response to get the memo name (ID)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var memoResp CreateMemoResponse
	if err := json.Unmarshal(bodyBytes, &memoResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Step 2: Update the displayTime via PATCH
	displayTime := createdTime.Format(time.RFC3339)
	updateReq := UpdateMemoRequest{
		DisplayTime: displayTime,
	}

	updateBody, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	// PATCH request to update displayTime
	patchURL := fmt.Sprintf("%s/api/v1/%s", c.baseURL, memoResp.Name)
	patchReq, err := http.NewRequest("PATCH", patchURL, bytes.NewReader(updateBody))
	if err != nil {
		return fmt.Errorf("failed to create patch request: %w", err)
	}

	patchReq.Header.Set("Authorization", "Bearer "+c.token)
	patchReq.Header.Set("Content-Type", "application/json")

	patchResp, err := c.httpClient.Do(patchReq)
	if err != nil {
		return fmt.Errorf("patch request failed: %w", err)
	}
	defer patchResp.Body.Close()

	if patchResp.StatusCode < 200 || patchResp.StatusCode >= 300 {
		patchBodyBytes, _ := io.ReadAll(patchResp.Body)
		return fmt.Errorf("patch request failed with status %d: %s", patchResp.StatusCode, string(patchBodyBytes))
	}

	fmt.Printf("DEBUG: Updated memo %s with displayTime: %s\n", memoResp.Name, displayTime)

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
