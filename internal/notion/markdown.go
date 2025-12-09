package notion

import (
	"fmt"
	"strings"
	"time"
)

// BlocksToMarkdown converts Notion blocks to Markdown format
func BlocksToMarkdown(blocks []Block, createdTime, pageTitle string, tags []string) (string, error) {
	var md strings.Builder

	// Add page title as H1
	if pageTitle != "" {
		md.WriteString("# " + pageTitle + "\n\n")
	}

	// Add tags if present
	if len(tags) > 0 {
		for _, tag := range tags {
			md.WriteString("#" + sanitizeTag(tag) + " ")
		}
		md.WriteString("\n\n")
	}

	// Add creation timestamp as metadata comment
	if createdTime != "" {
		parsedTime, err := time.Parse(time.RFC3339, createdTime)
		if err == nil {
			md.WriteString(fmt.Sprintf("<!-- Created: %s -->\n\n", parsedTime.Format("2006-01-02 15:04:05")))
		}
	}

	for _, block := range blocks {
		blockMd := blockToMarkdown(&block)
		if blockMd != "" {
			md.WriteString(blockMd)
			md.WriteString("\n")
		}
	}

	return strings.TrimSpace(md.String()), nil
}

// blockToMarkdown converts a single block to Markdown
func blockToMarkdown(block *Block) string {
	switch block.Type {
	case "paragraph":
		if block.Paragraph != nil {
			text := richTextToMarkdown(block.Paragraph.RichText)
			if text != "" {
				return text + "\n"
			}
			return ""
		}
	case "heading_1":
		if block.Heading1 != nil {
			text := richTextToMarkdown(block.Heading1.RichText)
			return "## " + text + "\n"
		}
	case "heading_2":
		if block.Heading2 != nil {
			text := richTextToMarkdown(block.Heading2.RichText)
			return "### " + text + "\n"
		}
	case "heading_3":
		if block.Heading3 != nil {
			text := richTextToMarkdown(block.Heading3.RichText)
			return "#### " + text + "\n"
		}
	case "bulleted_list_item":
		if block.BulletedList != nil {
			text := richTextToMarkdown(block.BulletedList.RichText)
			return "- " + text + "\n"
		}
	case "numbered_list_item":
		if block.NumberedList != nil {
			text := richTextToMarkdown(block.NumberedList.RichText)
			return "1. " + text + "\n"
		}
	case "to_do":
		if block.ToDo != nil {
			text := richTextToMarkdown(block.ToDo.RichText)
			checkbox := "- [ ]"
			if block.ToDo.Checked {
				checkbox = "- [x]"
			}
			return checkbox + " " + text + "\n"
		}
	case "code":
		if block.Code != nil {
			text := richTextToPlainText(block.Code.RichText)
			lang := block.Code.Language
			if lang == "" {
				lang = "text"
			}
			return fmt.Sprintf("```%s\n%s\n```\n", lang, text)
		}
	}
	return ""
}

// richTextToMarkdown converts rich text to Markdown with formatting
func richTextToMarkdown(richTexts []RichText) string {
	var result strings.Builder

	for _, rt := range richTexts {
		text := rt.PlainText
		if text == "" {
			continue
		}

		// Apply annotations
		if rt.Annotations != nil {
			if rt.Annotations.Code {
				text = "`" + text + "`"
			}
			if rt.Annotations.Bold {
				text = "**" + text + "**"
			}
			if rt.Annotations.Italic {
				text = "*" + text + "*"
			}
			if rt.Annotations.Strikethrough {
				text = "~~" + text + "~~"
			}
		}

		// Handle links
		if rt.Href != nil && *rt.Href != "" {
			text = fmt.Sprintf("[%s](%s)", text, *rt.Href)
		} else if rt.Text != nil && rt.Text.Link != nil {
			text = fmt.Sprintf("[%s](%s)", text, rt.Text.Link.URL)
		}

		result.WriteString(text)
	}

	return result.String()
}

// richTextToPlainText converts rich text to plain text without formatting
func richTextToPlainText(richTexts []RichText) string {
	var result strings.Builder
	for _, rt := range richTexts {
		result.WriteString(rt.PlainText)
	}
	return result.String()
}

// sanitizeTag removes spaces and special characters from tags
func sanitizeTag(tag string) string {
	// Replace spaces and dots with underscores
	tag = strings.ReplaceAll(tag, " ", "_")
	tag = strings.ReplaceAll(tag, ".", "_")
	// Remove any characters that aren't alphanumeric, underscore, or hyphen
	var result strings.Builder
	for _, r := range tag {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
