package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	notionAPIBase    = "https://api.notion.com/v1"
	notionAPIVersion = "2025-09-03"
	rateLimit        = 3 // 3 requests per second
)

// Client is a Notion API client
type Client struct {
	token      string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// NewClient creates a new Notion API client
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    rate.NewLimiter(rate.Limit(rateLimit), 1),
	}
}

// SearchResponse represents the response from the search API
type SearchResponse struct {
	Object     string  `json:"object"`
	Results    []Page  `json:"results"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// Page represents a Notion page
type Page struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	Parent         map[string]interface{} `json:"parent"`
	Properties     map[string]Property    `json:"properties"`
	URL            string                 `json:"url"`
}

// Property represents a page property
type Property struct {
	ID    string     `json:"id"`
	Type  string     `json:"type"`
	Title []RichText `json:"title,omitempty"`
}

// RichText represents rich text content
type RichText struct {
	Type        string       `json:"type"`
	Text        *TextContent `json:"text,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	PlainText   string       `json:"plain_text"`
	Href        *string      `json:"href,omitempty"`
}

// TextContent represents text content
type TextContent struct {
	Content string `json:"content"`
	Link    *Link  `json:"link,omitempty"`
}

// Link represents a link
type Link struct {
	URL string `json:"url"`
}

// Annotations represents text formatting
type Annotations struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

// BlockResponse represents the response from retrieving blocks
type BlockResponse struct {
	Object     string  `json:"object"`
	Results    []Block `json:"results"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// Block represents a Notion block
type Block struct {
	Object         string          `json:"object"`
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	CreatedTime    string          `json:"created_time"`
	LastEditedTime string          `json:"last_edited_time"`
	HasChildren    bool            `json:"has_children"`
	Paragraph      *ParagraphBlock `json:"paragraph,omitempty"`
	Heading1       *HeadingBlock   `json:"heading_1,omitempty"`
	Heading2       *HeadingBlock   `json:"heading_2,omitempty"`
	Heading3       *HeadingBlock   `json:"heading_3,omitempty"`
	BulletedList   *ListBlock      `json:"bulleted_list_item,omitempty"`
	NumberedList   *ListBlock      `json:"numbered_list_item,omitempty"`
	ToDo           *ToDoBlock      `json:"to_do,omitempty"`
	Code           *CodeBlock      `json:"code,omitempty"`
}

// ParagraphBlock represents a paragraph block
type ParagraphBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color"`
}

// HeadingBlock represents a heading block
type HeadingBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color"`
}

// ListBlock represents a list item block
type ListBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color"`
}

// ToDoBlock represents a to-do block
type ToDoBlock struct {
	RichText []RichText `json:"rich_text"`
	Checked  bool       `json:"checked"`
	Color    string     `json:"color"`
}

// CodeBlock represents a code block
type CodeBlock struct {
	RichText []RichText `json:"rich_text"`
	Language string     `json:"language"`
}

// doRequest performs an HTTP request with rate limiting
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", notionAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// SearchPages searches for pages matching the query
func (c *Client) SearchPages(query string) ([]Page, error) {
	var allPages []Page
	var cursor *string

	for {
		payload := map[string]interface{}{
			"page_size": 100,
		}
		if query != "" {
			payload["query"] = query
		}
		if cursor != nil {
			payload["start_cursor"] = *cursor
		}

		// Filter for pages only
		payload["filter"] = map[string]interface{}{
			"property": "object",
			"value":    "page",
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", notionAPIBase+"/search", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.doRequest(req)
		if err != nil {
			return nil, err
		}

		var searchResp SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		allPages = append(allPages, searchResp.Results...)

		if !searchResp.HasMore {
			break
		}
		cursor = searchResp.NextCursor
	}

	return allPages, nil
}

// RetrievePage retrieves a page by ID
func (c *Client) RetrievePage(pageID string) (*Page, error) {
	req, err := http.NewRequest("GET", notionAPIBase+"/pages/"+pageID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var page Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &page, nil
}

// RetrieveBlocks retrieves all blocks for a page or block
func (c *Client) RetrieveBlocks(blockID string) ([]Block, error) {
	var allBlocks []Block
	var cursor *string

	for {
		url := fmt.Sprintf("%s/blocks/%s/children?page_size=100", notionAPIBase, blockID)
		if cursor != nil {
			url += "&start_cursor=" + *cursor
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.doRequest(req)
		if err != nil {
			return nil, err
		}

		var blockResp BlockResponse
		if err := json.NewDecoder(resp.Body).Decode(&blockResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		allBlocks = append(allBlocks, blockResp.Results...)

		if !blockResp.HasMore {
			break
		}
		cursor = blockResp.NextCursor
	}

	return allBlocks, nil
}

// GetPageTitle extracts the title from a page
func (p *Page) GetPageTitle() string {
	for _, prop := range p.Properties {
		if prop.Type == "title" && len(prop.Title) > 0 {
			return prop.Title[0].PlainText
		}
	}
	return "Untitled"
}
