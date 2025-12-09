# notion2memos

A CLI tool to migrate your notes from Notion to Memos (https://usememos.com/).

## Features

- 🔄 **Full Migration**: Migrate all accessible Notion pages to Memos
- 🎯 **Title Filtering**: Filter pages by exact title match (supports multiple titles)
- ⏸️ **Resume Capability**: Resume interrupted migrations from where you left off
- 🧪 **Dry-Run Mode**: Preview migrations without actually creating memos
- 📝 **Markdown Support**: Converts Notion blocks to Markdown format
- ⏱️ **Timestamp Preservation**: Maintains original creation timestamps
- 🚦 **Rate Limiting**: Respects Notion's API rate limits (3 req/sec)
- ✅ **Supported Block Types**:
  - Paragraphs
  - Headings (H1, H2, H3)
  - Bulleted lists
  - Numbered lists
  - Checkboxes/To-do items
  - Code blocks

## Installation

```bash
go install github.com/OneManRepo/notion2memos@latest
```

Or clone and build from source:

```bash
git clone https://github.com/OneManRepo/notion2memos.git
cd notion2memos
go build -o notion2memos
```

## Setup

### 1. Create Notion Integration

1. Go to https://www.notion.so/my-integrations
2. Click "New integration"
3. Give it a name (e.g., "Notion2Memos")
4. Copy the "Internal Integration Token"
5. Share the Notion pages you want to migrate with this integration

### 2. Get Memos Access Token

1. Open your Memos instance
2. Go to Settings → Access Tokens
3. Create a new token
4. Copy the token

### 3. Configure notion2memos

Run the init command to create a configuration file:

```bash
notion2memos init
```

This creates `~/.notion2memos/config.yaml`. Edit it and add your tokens:

```yaml
notion_token: "secret_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
memos_url: "https://memos.example.com"
memos_token: "YOUR_MEMOS_ACCESS_TOKEN_HERE"
```

Alternatively, you can use environment variables:
- `NOTION_TOKEN`
- `MEMOS_URL`
- `MEMOS_TOKEN`

## Usage

### Basic Migration

Migrate all accessible Notion pages to Memos:

```bash
notion2memos migrate
```

### Dry-Run Mode

Preview what would be migrated without creating memos:

```bash
notion2memos migrate --dry-run
```

Markdown files will be saved to `./dry-run-output/`

### Filter by Title

Migrate only specific pages by exact title match:

```bash
notion2memos migrate --filter-title "My Note" --filter-title "Another Note"
```

### Resume Migration

If a migration is interrupted, resume from where it left off:

```bash
notion2memos migrate --resume
```

### Reset State

Clear the migration state to start fresh:

```bash
notion2memos reset
```

### Custom Config File

Use a custom configuration file:

```bash
notion2memos migrate --config /path/to/config.yaml
```

## Commands

- `notion2memos init` - Create configuration file template
- `notion2memos migrate` - Migrate pages from Notion to Memos
- `notion2memos reset` - Reset migration state
- `notion2memos version` - Print version number
- `notion2memos export` - Export from Notion (not yet implemented)
- `notion2memos import` - Import to Memos (not yet implemented)

## Configuration

Configuration can be provided via:
1. Config file: `~/.notion2memos/config.yaml` (default) or `./config.yaml`
2. Custom config: `--config /path/to/config.yaml`
3. Environment variables: `NOTION_TOKEN`, `MEMOS_URL`, `MEMOS_TOKEN`

See `config.example.yaml` for a complete example.

## Migration State

The tool tracks processed pages in `~/.notion2memos/state.json` to support resuming. Use `notion2memos reset` to clear this state.

## Limitations

- Only migrates simple text notes with basic formatting
- Nested pages are treated as separate pages
- Databases are not fully supported
- Some Notion block types are not yet implemented (images, embeds, etc.)
- Requires pages to be explicitly shared with the integration

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

See LICENSE file for details.
