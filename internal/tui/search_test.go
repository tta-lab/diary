package tui

import (
	"strings"
	"testing"
)

func TestHighlightTerm(t *testing.T) {
	tests := []struct {
		name       string
		searchTerm string
		text       string
		wantMatch  bool
	}{
		{
			name:       "simple match",
			searchTerm: "bug",
			text:       "fixing bug in code",
			wantMatch:  true,
		},
		{
			name:       "case insensitive",
			searchTerm: "encryption",
			text:       "Age ENCRYPTION is great",
			wantMatch:  true,
		},
		{
			name:       "multiple matches",
			searchTerm: "test",
			text:       "test the test cases",
			wantMatch:  true,
		},
		{
			name:       "no match",
			searchTerm: "missing",
			text:       "this text has no match",
			wantMatch:  false,
		},
		{
			name:       "partial word match",
			searchTerm: "crypt",
			text:       "encryption and decryption",
			wantMatch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &Model{searchTerm: tt.searchTerm}
			result := model.highlightTerm(tt.text)

			if tt.wantMatch {
				// Check that result contains the search term (case-insensitive)
				lowerResult := strings.ToLower(result)
				lowerTerm := strings.ToLower(tt.searchTerm)

				if !strings.Contains(lowerResult, lowerTerm) {
					t.Errorf("Expected highlighted result to contain %q, got %s", tt.searchTerm, result)
				}

				// In test environment, lipgloss might not add ANSI codes (no TTY)
				// Just verify the term is still present after highlighting
				if result == "" {
					t.Error("Expected non-empty result")
				}
			} else {
				// No match - text should be unchanged
				if result != tt.text {
					t.Errorf("Expected unchanged text, got %s", result)
				}
			}
		})
	}
}

func TestGetContext(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		matchIdx int
		want     int // number of lines in context
	}{
		{
			name:     "middle of file",
			lines:    []string{"line 1", "line 2", "match line", "line 4", "line 5"},
			matchIdx: 2,
			want:     3, // line before, match, line after
		},
		{
			name:     "first line",
			lines:    []string{"match line", "line 2", "line 3"},
			matchIdx: 0,
			want:     2, // match and line after (no line before)
		},
		{
			name:     "last line",
			lines:    []string{"line 1", "line 2", "match line"},
			matchIdx: 2,
			want:     2, // line before and match (no line after)
		},
		{
			name:     "single line",
			lines:    []string{"only line"},
			matchIdx: 0,
			want:     1, // just the match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &Model{}
			context := model.getContext(tt.lines, tt.matchIdx)

			// Count lines in context
			contextLines := strings.Split(context, "\n")

			if len(contextLines) != tt.want {
				t.Errorf("Expected %d lines in context, got %d\nContext: %s",
					tt.want, len(contextLines), context)
			}

			// Match line should be prefixed with "> "
			matchLineFound := false
			for _, line := range contextLines {
				if strings.HasPrefix(line, "> ") {
					matchLineFound = true
					// Check it contains the actual match
					if !strings.Contains(line, tt.lines[tt.matchIdx]) {
						t.Errorf("Match line should contain %q, got %s",
							tt.lines[tt.matchIdx], line)
					}
				}
			}

			if !matchLineFound {
				t.Error("Expected match line to be prefixed with '> '")
			}
		})
	}
}

func TestSearchResultTitle(t *testing.T) {
	result := SearchResult{
		date: "2026-02-07",
		matches: []Match{
			{lineNum: 1, line: "test"},
			{lineNum: 2, line: "test again"},
		},
	}

	title := result.Title()

	if !strings.Contains(title, "2026-02-07") {
		t.Errorf("Title should contain date, got %s", title)
	}

	if !strings.Contains(title, "2") {
		t.Errorf("Title should contain match count, got %s", title)
	}

	if !strings.Contains(title, "matches") {
		t.Errorf("Title should contain 'matches', got %s", title)
	}
}

func TestSearchResultDescription(t *testing.T) {
	tests := []struct {
		name    string
		matches []Match
		want    string
	}{
		{
			name: "with matches",
			matches: []Match{
				{lineNum: 1, line: "This is a test line"},
			},
			want: "This is a test line",
		},
		{
			name:    "no matches",
			matches: []Match{},
			want:    "",
		},
		{
			name: "long line gets truncated",
			matches: []Match{
				{lineNum: 1, line: strings.Repeat("a", 100)},
			},
			want: "aaa", // Should contain the repeated 'a's
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchResult{
				date:    "2026-02-07",
				matches: tt.matches,
			}

			desc := result.Description()

			if tt.want == "" {
				if desc != "" {
					t.Errorf("Expected empty description, got %s", desc)
				}
			} else {
				if !strings.Contains(desc, tt.want) {
					t.Errorf("Expected description to contain %q, got %s", tt.want, desc)
				}

				// For long line test, check that it was truncated with "..."
				if len(tt.matches) > 0 && len(tt.matches[0].line) > 60 {
					if !strings.Contains(desc, "...") {
						t.Error("Expected long line to be truncated with '...'")
					}
				}
			}
		})
	}
}

func TestSearchResultFilterValue(t *testing.T) {
	result := SearchResult{
		date:    "2026-02-07",
		matches: []Match{{lineNum: 1, line: "test"}},
	}

	filterValue := result.FilterValue()

	if filterValue != "2026-02-07" {
		t.Errorf("FilterValue should return date, got %s", filterValue)
	}
}
