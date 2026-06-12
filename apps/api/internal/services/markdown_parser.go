package services

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// BatchProblemDraft is a single problem parsed from a batch markdown file.
type BatchProblemDraft struct {
	Slug          string            `yaml:"slug" json:"slug"`
	Title         string            `yaml:"title" json:"title"`
	TimeLimitMS   int               `yaml:"time_limit_ms" json:"time_limit_ms"`
	MemoryLimitMB int               `yaml:"memory_limit_mb" json:"memory_limit_mb"`
	OutputLimitKB int               `yaml:"output_limit_kb" json:"output_limit_kb"`
	Label         string            `yaml:"label" json:"label"`
	Score         int               `yaml:"score" json:"score"`
	Statement     string            `yaml:"-" json:"statement"`
	Cases         []ProblemCaseDraft `yaml:"-" json:"cases"`
	Assets        []ProblemAssetDraft `yaml:"-" json:"assets,omitempty"`
}

// BatchMarkdownResult holds all parsed problems from a markdown file.
type BatchMarkdownResult struct {
	Problems []BatchProblemDraft `json:"problems"`
	Warnings []string            `json:"warnings,omitempty"`
}

var (
	htmlTagRe       = regexp.MustCompile(`<[^>]*>`)
	htmlEntityRe    = regexp.MustCompile(`&[a-zA-Z]+;`)
	frontmatterRe   = regexp.MustCompile(`^---\s*\n([\s\S]*?)\n---\s*\n`)
	codeBlockRe     = regexp.MustCompile("```(input|output)\\s*\\n([\\s\\S]*?)```")
	sectionSplitRe  = regexp.MustCompile(`\n---\s*\n`)
	multiNewlineRe  = regexp.MustCompile(`\n{4,}`)
	markdownImageRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
)

// ParseBatchMarkdown parses a .md file containing multiple problems
// separated by YAML frontmatter blocks.
func ParseBatchMarkdown(raw string) (*BatchMarkdownResult, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	// Strip UTF-8 BOM
	raw = strings.TrimPrefix(raw, "\ufeff")

	result := &BatchMarkdownResult{
		Problems: make([]BatchProblemDraft, 0),
		Warnings: make([]string, 0),
	}

	// Split by "---" on its own line (rough sections)
	sections := splitMarkdownSections(raw)

	for i, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		draft, warnings := parseProblemSection(section, i+1)
		if draft == nil {
			continue
		}
		result.Problems = append(result.Problems, *draft)
		result.Warnings = append(result.Warnings, warnings...)
	}

	if len(result.Problems) == 0 {
		return nil, &ParseError{Message: "no valid problem sections found in markdown"}
	}

	return result, nil
}

// splitMarkdownSections splits raw markdown by "\n---\n" separators
// but only when "---" is on its own line (no other content on the line).
func splitMarkdownSections(raw string) []string {
	// Strategy: find all positions of "\n---\n" and use them as boundaries
	// But we need to handle the very first "---" (frontmatter opener) differently.
	// A problem section looks like:
	//   ---
	//   slug: foo
	//   ---
	//   statement...
	//   ---
	//   slug: bar
	//   ---
	//   statement...

	lines := strings.Split(raw, "\n")
	sections := make([]string, 0)
	currentStart := 0
	inFrontmatter := false
	frontmatterEnded := false

	for i, line := range lines {
		isHR := strings.TrimSpace(line) == "---"

		if isHR && !inFrontmatter && !frontmatterEnded {
			// Opening frontmatter delimiter
			inFrontmatter = true
			continue
		}

		if isHR && inFrontmatter {
			// Closing frontmatter delimiter
			inFrontmatter = false
			frontmatterEnded = true
			currentStart = i + 1 // statement starts after closing ---
			continue
		}

		if isHR && frontmatterEnded {
			// This is a separator between problems
			// End current section, start new one
			section := strings.Join(lines[currentStart:i], "\n")
			sections = append(sections, section)
			currentStart = i + 1
			inFrontmatter = true   // next line could be frontmatter
			frontmatterEnded = false
			continue
		}
	}

	// Last section
	if currentStart < len(lines) {
		section := strings.Join(lines[currentStart:], "\n")
		sections = append(sections, section)
	}

	return sections
}

// parseProblemSection parses a single problem section (frontmatter + statement).
func parseProblemSection(section string, index int) (*BatchProblemDraft, []string) {
	warnings := make([]string, 0)

	// Check if section starts with frontmatter
	fm := frontmatterRe.FindStringSubmatch(section)
	if fm == nil {
		// No frontmatter - treat entire section as statement
		// Need at least a title (use first heading)
		title := extractFirstHeading(section)
		if title == "" {
			warnings = append(warnings, "section has no frontmatter or heading, skipping")
			return nil, warnings
		}
		slug := slugify(title)
		return &BatchProblemDraft{
			Slug:          slug,
			Title:         title,
			Statement:     strings.TrimSpace(section),
			TimeLimitMS:   1000,
			MemoryLimitMB: 256,
			OutputLimitKB: 1024,
			Score:         100,
			Label:         defaultLabel(index),
		}, warnings
	}

	frontmatterRaw := fm[1]
	statement := strings.TrimSpace(section[len(fm[0]):])

	// Parse YAML frontmatter
	var draft BatchProblemDraft
	if err := yaml.Unmarshal([]byte(frontmatterRaw), &draft); err != nil {
		warnings = append(warnings, "failed to parse frontmatter: "+err.Error())
		// Try to salvage: extract title from statement
		title := extractFirstHeading(statement)
		if title == "" {
			return nil, warnings
		}
		draft.Title = title
		draft.Slug = slugify(title)
	}

	// Apply defaults
	if draft.Slug == "" {
		draft.Slug = slugify(draft.Title)
	}
	if draft.Title == "" {
		draft.Title = extractFirstHeading(statement)
		if draft.Title == "" {
			warnings = append(warnings, "no title found, skipping")
			return nil, warnings
		}
	}
	if draft.TimeLimitMS <= 0 {
		draft.TimeLimitMS = 1000
	}
	if draft.MemoryLimitMB <= 0 {
		draft.MemoryLimitMB = 256
	}
	if draft.OutputLimitKB <= 0 {
		draft.OutputLimitKB = 1024
	}
	if draft.Score <= 0 {
		draft.Score = 100
	}
	if draft.Label == "" {
		draft.Label = defaultLabel(index)
	}

	// Process statement: strip HTML, extract test cases
	cleanStatement, cases := extractTestCases(statement)
	cleanStatement = sanitizeMarkdown(cleanStatement)
	cleanStatement = extractImagesAsAssets(cleanStatement)
	draft.Statement = strings.TrimSpace(cleanStatement)
	draft.Cases = cases

	return &draft, warnings
}

// extractTestCases finds embedded test cases in the markdown statement.
// Test cases are defined as paired ```input and ```output code blocks.
func extractTestCases(statement string) (string, []ProblemCaseDraft) {
	cases := make([]ProblemCaseDraft, 0)
	caseIndex := 0

	// Find all code blocks with input/output tags
	cleaned := codeBlockRe.ReplaceAllStringFunc(statement, func(match string) string {
		submatches := codeBlockRe.FindStringSubmatch(match)
		if submatches == nil {
			return match
		}
		blockType := submatches[1]
		content := strings.TrimSpace(submatches[2])

		if blockType == "input" {
			caseIndex++
			cases = append(cases, ProblemCaseDraft{
				Name:   "",
				Input:  content,
				Output: "",
				Weight: 0,
			})
			return "" // Remove from statement
		}

		// blockType == "output"
		if caseIndex > 0 && caseIndex <= len(cases) {
			cases[caseIndex-1].Output = content
		}
		return "" // Remove from statement
	})

	// Filter out incomplete case pairs (missing input or output)
	complete := make([]ProblemCaseDraft, 0, len(cases))
	for i, c := range cases {
		if c.Input != "" && c.Output != "" {
			c.Name = ""
			c.Weight = 0
			complete = append(complete, c)
		}
		_ = i
	}

	return cleaned, complete
}

// sanitizeMarkdown cleans up the markdown for use as a problem statement.
func sanitizeMarkdown(raw string) string {
	// Remove raw HTML tags (but preserve content inside them)
	raw = htmlTagRe.ReplaceAllString(raw, "")

	// Decode common HTML entities
	raw = htmlEntityRe.ReplaceAllStringFunc(raw, func(entity string) string {
		switch entity {
		case "&amp;":
			return "&"
		case "&lt;":
			return "<"
		case "&gt;":
			return ">"
		case "&quot;":
			return "\""
		case "&apos;":
			return "'"
		case "&nbsp;":
			return " "
		default:
			return entity
		}
	})

	// Collapse excessive blank lines
	raw = multiNewlineRe.ReplaceAllString(raw, "\n\n\n")

	return strings.TrimSpace(raw)
}

// extractImagesAsAssets identifies markdown images and notes them for asset handling.
// Images with relative paths are flagged; external URLs are left as-is.
func extractImagesAsAssets(raw string) string {
	// Markdown images like ![alt](path) are left in the statement.
	// The renderer resolves relative paths at display time.
	// We just note them here for potential asset extraction.
	_ = markdownImageRe.FindAllStringSubmatch(raw, -1)
	return raw
}

// extractFirstHeading returns the content of the first markdown heading.
func extractFirstHeading(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

// slugify converts a title to a URL-friendly slug.
func slugify(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	result := make([]rune, 0, len(slug))
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			result = append(result, r)
		}
	}
	slug = string(result)
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "problem"
	}
	return slug
}

// defaultLabel returns the default problem label for the given index.
func defaultLabel(index int) string {
	idx := index
	label := ""
	for {
		label = string(rune('A'+idx%26)) + label
		idx = idx/26 - 1
		if idx < 0 {
			break
		}
	}
	return label
}

// ParseError represents a markdown parsing error.
type ParseError struct {
	Message string
}

func (e *ParseError) Error() string {
	return e.Message
}
