package migrate

import (
"fmt"
"log"
"time"

"github.com/OneManRepo/notion2memos/internal/config"
"github.com/OneManRepo/notion2memos/internal/memos"
"github.com/OneManRepo/notion2memos/internal/notion"
"github.com/schollz/progressbar/v3"
)

// Migrator coordinates the migration from Notion to Memos
type Migrator struct {
	notionClient *notion.Client
	memosClient  *memos.Client
	state        *config.State
	dryRun       bool
}

// NewMigrator creates a new Migrator
func NewMigrator(cfg *config.Config, dryRun bool) (*Migrator, error) {
	state, err := config.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return &Migrator{
		notionClient: notion.NewClient(cfg.NotionToken),
		memosClient:  memos.NewClient(cfg.MemosURL, cfg.MemosToken),
		state:        state,
		dryRun:       dryRun,
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

	// Convert blocks to Markdown
	markdown, err := notion.BlocksToMarkdown(blocks, page.CreatedTime)
	if err != nil {
		return fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// If content is empty, skip
	if markdown == "" {
		log.Printf("Skipping empty page: %s\n", page.GetPageTitle())
		return nil
	}

	// Parse created time
	createdTime, err := time.Parse(time.RFC3339, page.CreatedTime)
	if err != nil {
		// Fallback to current time if parsing fails
		createdTime = time.Now()
	}

	// Create memo in Memos
	if err := m.memosClient.CreateMemo(markdown, createdTime, m.dryRun); err != nil {
		return fmt.Errorf("failed to create memo: %w", err)
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
