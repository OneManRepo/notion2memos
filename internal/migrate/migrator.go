package migrate

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/OneManRepo/notion2memos/internal/config"
	"github.com/OneManRepo/notion2memos/internal/memos"
	"github.com/OneManRepo/notion2memos/internal/notion"
	"github.com/schollz/progressbar/v3"
)

// Migrator coordinates the migration from Notion to Memos
type Migrator struct {
	notionClient  *notion.Client
	memosClient   *memos.Client
	state         *config.State
	dryRun        bool
	pageCache     map[string]*notion.Page
	databaseCache map[string]*notion.Database
}

// NewMigrator creates a new Migrator
func NewMigrator(cfg *config.Config, dryRun bool) (*Migrator, error) {
	state, err := config.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return &Migrator{
		notionClient:  notion.NewClient(cfg.NotionToken),
		memosClient:   memos.NewClient(cfg.MemosURL, cfg.MemosToken),
		state:         state,
		dryRun:        dryRun,
		pageCache:     make(map[string]*notion.Page),
		databaseCache: make(map[string]*notion.Database),
	}, nil
}

// MigrateOptions contains options for migration
type MigrateOptions struct {
	Resume       bool
	FilterTitles []string
}

// Migrate performs the migration from Notion to Memos
func (m *Migrator) Migrate(opts MigrateOptions) error {
	log.Println("Starting migration from Notion to Memos...")

	if m.dryRun {
		log.Println("DRY RUN MODE: Memos will be saved to ./dry-run-output/ instead of being created")
	}

	// Search for all pages
	log.Println("Searching for pages in Notion...")
	pages, err := m.notionClient.SearchPages("")
	if err != nil {
		return fmt.Errorf("failed to search pages: %w", err)
	}

	log.Printf("Found %d pages\n", len(pages))

	// Filter pages if titles are specified
	if len(opts.FilterTitles) > 0 {
		pages = m.filterPagesByTitle(pages, opts.FilterTitles)
		log.Printf("Filtered to %d pages matching specified titles\n", len(pages))
	}

	// Filter out already processed pages if resuming
	if opts.Resume {
		originalCount := len(pages)
		pages = m.filterProcessedPages(pages)
		skipped := originalCount - len(pages)
		if skipped > 0 {
			log.Printf("Skipping %d already processed pages (resume mode)\n", skipped)
		}
	}

	if len(pages) == 0 {
		log.Println("No pages to migrate")
		return nil
	}

	// Create progress bar
	bar := progressbar.Default(int64(len(pages)), "Migrating pages")

	// Process each page
	successCount := 0
	for _, page := range pages {
		if err := m.migratePage(&page); err != nil {
			bar.Close()
			return fmt.Errorf("failed to migrate page %s (%s): %w", page.GetPageTitle(), page.ID, err)
		}

		// Mark as processed and save state
		m.state.MarkProcessed(page.ID)
		if err := m.state.SaveState(); err != nil {
			bar.Close()
			return fmt.Errorf("failed to save state: %w", err)
		}

		successCount++
		bar.Add(1)
	}

	bar.Finish()
	log.Printf("\nMigration completed successfully! Migrated %d pages\n", successCount)

	if m.dryRun {
		log.Println("Check ./dry-run-output/ for the generated markdown files")
	}

	return nil
}

// migratePage migrates a single page from Notion to Memos
func (m *Migrator) migratePage(page *notion.Page) error {
	// Retrieve page blocks
	blocks, err := m.notionClient.RetrieveBlocks(page.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve blocks: %w", err)
	}

	// Get page title
	pageTitle := page.GetPageTitle()

	// Skip pages with no content blocks
	if len(blocks) == 0 {
		log.Printf("Skipping empty page (no blocks): %s\n", pageTitle)
		return nil
	}

	// Get parent tags (using cache)
	tags, err := m.getParentTagsCached(page)
	if err != nil {
		// Log warning but continue - tags are not critical
		log.Printf("Warning: failed to retrieve parent tags for page %s: %v\n", page.GetPageTitle(), err)
	}

	// Replace "Tagebuch" tag with "tagebuch" (lowercase)
	for i, tag := range tags {
		if tag == "Tagebuch" {
			tags[i] = "tagebuch"
			break
		}
	}

	// Convert blocks to Markdown with title and tags
	markdown, err := notion.BlocksToMarkdown(blocks, page.CreatedTime, pageTitle, tags)
	if err != nil {
		return fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// If content is empty after conversion, skip
	if markdown == "" {
		log.Printf("Skipping empty page: %s\n", pageTitle)
		return nil
	}

	// Parse created time from Notion (RFC3339 format)
	createdTime, err := time.Parse(time.RFC3339, page.CreatedTime)
	if err != nil {
		log.Printf("WARNING: Failed to parse created time for page '%s': %v. Using current time as fallback.\n", pageTitle, err)
		createdTime = time.Now()
	}

	// Check if content exceeds Memos API limit and split if necessary
	const memosMaxLength = 8192
	if len(markdown) > memosMaxLength {
		log.Printf("Page '%s' exceeds character limit (%d chars). Splitting into multiple memos...\n", pageTitle, len(markdown))
		if err := m.createSplitMemos(markdown, pageTitle, createdTime); err != nil {
			return fmt.Errorf("failed to create split memos: %w", err)
		}
	} else {
		// Create single memo in Memos
		if err := m.memosClient.CreateMemo(markdown, createdTime, m.dryRun); err != nil {
			return fmt.Errorf("failed to create memo: %w", err)
		}
	}

	return nil
}

// createSplitMemos splits a long memo into multiple parts and creates them
func (m *Migrator) createSplitMemos(content, pageTitle string, createdTime time.Time) error {
	const memosMaxLength = 8192
	const splitMarker = "\n\n..."
	const continuationMarker = "...\n\n"

	// Calculate safe chunk size (leaving room for markers and title modification)
	const safeChunkSize = memosMaxLength - 200 // Reserve space for title, tags, and markers

	var parts []string
	remaining := content

	// Split content into chunks
	for len(remaining) > safeChunkSize {
		// Find a good breaking point (end of line) before the chunk size
		breakPoint := safeChunkSize
		for breakPoint > 0 && remaining[breakPoint] != '\n' {
			breakPoint--
		}

		// If no newline found, just break at chunk size
		if breakPoint == 0 {
			breakPoint = safeChunkSize
		}

		parts = append(parts, remaining[:breakPoint])
		remaining = remaining[breakPoint:]

		// Trim leading whitespace from remaining content
		for len(remaining) > 0 && (remaining[0] == '\n' || remaining[0] == ' ') {
			remaining = remaining[1:]
		}
	}

	// Add the final part
	if len(remaining) > 0 {
		parts = append(parts, remaining)
	}

	log.Printf("Split page '%s' into %d parts\n", pageTitle, len(parts))

	// Create each part as a separate memo
	for i, part := range parts {
		partNumber := i + 1
		partTitle := fmt.Sprintf("%s (%d/%d)", pageTitle, partNumber, len(parts))

		// Replace the original title with the numbered title
		lines := strings.Split(part, "\n")
		var contentWithoutTitle string
		startIndex := 0

		// Find and remove the original title if present
		if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
			startIndex = 1
		}
		contentWithoutTitle = strings.Join(lines[startIndex:], "\n")

		// Build the memo content with markers
		var memoContent string
		if i == 0 {
			// First part: title + content + "..."
			memoContent = "# " + partTitle + "\n\n" + contentWithoutTitle + splitMarker
		} else if i == len(parts)-1 {
			// Last part: title + "..." + content
			memoContent = "# " + partTitle + "\n\n" + continuationMarker + contentWithoutTitle
		} else {
			// Middle parts: title + "..." + content + "..."
			memoContent = "# " + partTitle + "\n\n" + continuationMarker + contentWithoutTitle + splitMarker
		}

		// Offset creation time by a few seconds for each part
		partCreatedTime := createdTime.Add(time.Duration(i*5) * time.Second)

		// Create the memo
		if err := m.memosClient.CreateMemo(memoContent, partCreatedTime, m.dryRun); err != nil {
			return fmt.Errorf("failed to create memo part %d: %w", partNumber, err)
		}

		log.Printf("Created memo part %d/%d for page '%s'\n", partNumber, len(parts), pageTitle)
	}

	return nil
}

// filterPagesByTitle filters pages to only include those with matching titles
func (m *Migrator) filterPagesByTitle(pages []notion.Page, titles []string) []notion.Page {
	if len(titles) == 0 {
		return pages
	}

	titleSet := make(map[string]bool)
	for _, title := range titles {
		titleSet[title] = true
	}

	var filtered []notion.Page
	for _, page := range pages {
		if titleSet[page.GetPageTitle()] {
			filtered = append(filtered, page)
		}
	}

	return filtered
}

// filterProcessedPages removes already processed pages
func (m *Migrator) filterProcessedPages(pages []notion.Page) []notion.Page {
	var filtered []notion.Page
	for _, page := range pages {
		if !m.state.IsProcessed(page.ID) {
			filtered = append(filtered, page)
		}
	}
	return filtered
}

// getPageCached retrieves a page with caching
func (m *Migrator) getPageCached(pageID string) (*notion.Page, error) {
	if cached, ok := m.pageCache[pageID]; ok {
		return cached, nil
	}

	page, err := m.notionClient.RetrievePage(pageID)
	if err != nil {
		return nil, err
	}

	m.pageCache[pageID] = page
	return page, nil
}

// getDatabaseCached retrieves a database with caching
func (m *Migrator) getDatabaseCached(databaseID string) (*notion.Database, error) {
	if cached, ok := m.databaseCache[databaseID]; ok {
		return cached, nil
	}

	database, err := m.notionClient.RetrieveDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	m.databaseCache[databaseID] = database
	return database, nil
}

// getParentTagsCached retrieves parent tags with caching
func (m *Migrator) getParentTagsCached(page *notion.Page) ([]string, error) {
	var tags []string

	// Check if parent is a database
	if dbID := page.GetParentDatabaseID(); dbID != "" {
		database, err := m.getDatabaseCached(dbID)
		if err == nil {
			tags = append(tags, database.GetDatabaseTitle())
		}
	}

	// Walk up the page parent chain (max 10 levels to prevent infinite loops)
	currentPageID := page.GetParentPageID()
	for i := 0; i < 10 && currentPageID != ""; i++ {
		parentPage, err := m.getPageCached(currentPageID)
		if err != nil {
			// If we can't retrieve the parent, just return what we have
			break
		}

		tags = append([]string{parentPage.GetPageTitle()}, tags...) // Prepend to maintain hierarchy
		currentPageID = parentPage.GetParentPageID()
	}

	return tags, nil
}
