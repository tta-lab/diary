package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/storage"
)

const (
	listHeight   = 10
	previewRatio = 0.6 // Preview takes 60% of height
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginLeft(2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238"))
)

// SearchResult represents search results for a single diary entry
type SearchResult struct {
	date    string
	matches []Match
}

// Match represents a single match within an entry
type Match struct {
	lineNum int
	line    string
	context string // surrounding context
}

// Title implements list.Item
func (r SearchResult) Title() string {
	return fmt.Sprintf("%s (%d matches)", r.date, len(r.matches))
}

// Description implements list.Item
func (r SearchResult) Description() string {
	if len(r.matches) == 0 {
		return ""
	}
	// Show first match as preview
	preview := r.matches[0].line
	if len(preview) > 60 {
		preview = preview[:60] + "..."
	}
	return dimStyle.Render(preview)
}

// FilterValue implements list.Item
func (r SearchResult) FilterValue() string {
	return r.date
}

// Model is the bubbletea model for search TUI
type Model struct {
	user          string
	searchTerm    string
	results       []SearchResult
	list          list.Model
	viewport      viewport.Model
	searchInput   textinput.Model
	inputMode     bool   // True when user is typing search term
	ready         bool
	width         int
	height        int
	keyPath       string
	quitting      bool
	openDate      string // Set when user wants to open full entry
	initialIndex  int    // Initial selection index to restore
}

type searchCompleteMsg struct {
	results []SearchResult
}

// NewSearchModel creates a new search TUI model
func NewSearchModel(user, searchTerm, keyPath string) Model {
	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Enter search term..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	// If no search term provided, start in input mode
	inputMode := searchTerm == ""
	if inputMode {
		ti.SetValue("")
	} else {
		ti.SetValue(searchTerm)
		ti.Blur()
	}

	return Model{
		user:         user,
		searchTerm:   searchTerm,
		keyPath:      keyPath,
		list:         list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		viewport:     viewport.New(0, 0),
		searchInput:  ti,
		inputMode:    inputMode,
		initialIndex: 0,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.inputMode {
		// Start in input mode, wait for user to enter term
		return textinput.Blink
	}
	// Start search immediately if term provided
	return m.performSearch
}

// performSearch searches all diary entries in parallel
func (m Model) performSearch() tea.Msg {
	entries, err := storage.ListEntries(m.user)
	if err != nil {
		return searchCompleteMsg{results: []SearchResult{}}
	}

	// Search in parallel
	resultChan := make(chan SearchResult, len(entries))
	var wg sync.WaitGroup

	for _, date := range entries {
		wg.Add(1)
		go func(date string) {
			defer wg.Done()
			result := m.searchEntry(date)
			if len(result.matches) > 0 {
				resultChan <- result
			}
		}(date)
	}

	// Wait and close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []SearchResult
	for result := range resultChan {
		results = append(results, result)
	}

	// Sort by date descending (newest first)
	// Simple sort: just reverse if needed
	return searchCompleteMsg{results: results}
}

// searchEntry searches a single diary entry
func (m Model) searchEntry(date string) SearchResult {
	result := SearchResult{date: date, matches: []Match{}}

	// Get entry path
	entryPath, err := storage.DiaryPath(m.user, date)
	if err != nil {
		return result
	}

	// Read encrypted content
	encrypted, err := storage.ReadEntry(entryPath)
	if err != nil {
		return result
	}

	// Decrypt
	plaintext, err := crypto.Decrypt(encrypted, m.keyPath)
	if err != nil {
		return result
	}

	// Search line by line
	lines := strings.Split(string(plaintext), "\n")
	searchLower := strings.ToLower(m.searchTerm)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), searchLower) {
			result.matches = append(result.matches, Match{
				lineNum: i + 1,
				line:    line,
				context: m.getContext(lines, i),
			})
		}
	}

	return result
}

// getContext returns surrounding lines for context
func (m Model) getContext(lines []string, matchIdx int) string {
	start := matchIdx - 1
	end := matchIdx + 1

	if start < 0 {
		start = 0
	}
	if end >= len(lines) {
		end = len(lines) - 1
	}

	var context []string
	for i := start; i <= end; i++ {
		prefix := "  "
		if i == matchIdx {
			prefix = "> "
		}
		context = append(context, prefix+lines[i])
	}

	return strings.Join(context, "\n")
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.list.SetSize(msg.Width, listHeight)
			m.viewport = viewport.New(msg.Width-4, msg.Height-listHeight-6)
			m.viewport.Style = borderStyle
			m.ready = true
		} else {
			m.list.SetSize(msg.Width, listHeight)
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - listHeight - 6
		}

	case searchCompleteMsg:
		m.results = msg.results
		items := make([]list.Item, len(m.results))
		for i, r := range m.results {
			items[i] = r
		}
		m.list.SetItems(items)
		m.list.Title = fmt.Sprintf("Search: \"%s\" (%d entries)", m.searchTerm, len(m.results))

		// Restore previous selection if set
		if m.initialIndex > 0 && m.initialIndex < len(m.results) {
			m.list.Select(m.initialIndex)
		}

		// Update preview for selected item
		if len(m.results) > 0 {
			m.updatePreview()
		}

	case tea.KeyMsg:
		// Handle input mode separately
		if m.inputMode {
			switch msg.String() {
			case "enter":
				// Start search with entered term
				m.searchTerm = m.searchInput.Value()
				if m.searchTerm != "" {
					m.inputMode = false
					m.searchInput.Blur()
					return m, m.performSearch
				}
			case "esc":
				if m.searchTerm == "" {
					// No previous search, quit
					m.quitting = true
					return m, tea.Quit
				}
				// Cancel input, return to results
				m.inputMode = false
				m.searchInput.Blur()
				m.searchInput.SetValue(m.searchTerm)
			}
		} else {
			// Results mode
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit

			case "esc":
				// Esc quits from results
				m.quitting = true
				return m, tea.Quit

			case "/":
				// Enter input mode for new search
				m.inputMode = true
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				return m, textinput.Blink

			case "enter":
				// Open selected entry in glow
				if len(m.results) > 0 {
					selected := m.list.SelectedItem()
					if result, ok := selected.(SearchResult); ok {
						m.openDate = result.date
						return m, tea.Quit
					}
				}
			}
		}
	}

	// Update search input if in input mode
	if m.inputMode {
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		// Update list only when not in input mode
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Update preview when selection changes
		if len(m.results) > 0 {
			m.updatePreview()
		}

		// Update viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updatePreview updates the preview pane with the selected entry's matches
func (m *Model) updatePreview() {
	if len(m.results) == 0 {
		m.viewport.SetContent("No results found")
		return
	}

	selected := m.list.SelectedItem()
	if result, ok := selected.(SearchResult); ok {
		var preview strings.Builder

		preview.WriteString(titleStyle.Render(fmt.Sprintf("📅 %s", result.date)))
		preview.WriteString("\n\n")

		for i, match := range result.matches {
			if i > 0 {
				preview.WriteString("\n")
			}

			// Show line number
			preview.WriteString(dimStyle.Render(fmt.Sprintf("Line %d:", match.lineNum)))
			preview.WriteString("\n")

			// Highlight the search term
			highlighted := m.highlightTerm(match.line)
			preview.WriteString(highlighted)
			preview.WriteString("\n")

			// Limit to first 5 matches to avoid clutter
			if i >= 4 && len(result.matches) > 5 {
				remaining := len(result.matches) - 5
				preview.WriteString(dimStyle.Render(fmt.Sprintf("\n... and %d more matches", remaining)))
				break
			}
		}

		preview.WriteString("\n\n")
		preview.WriteString(dimStyle.Render("Press Enter to view full entry in glow"))

		m.viewport.SetContent(preview.String())
	}
}

// highlightTerm highlights the search term in the text
func (m *Model) highlightTerm(text string) string {
	searchLower := strings.ToLower(m.searchTerm)
	textLower := strings.ToLower(text)

	var result strings.Builder
	lastIdx := 0

	for {
		idx := strings.Index(textLower[lastIdx:], searchLower)
		if idx == -1 {
			result.WriteString(text[lastIdx:])
			break
		}

		// Add text before match
		actualIdx := lastIdx + idx
		result.WriteString(text[lastIdx:actualIdx])

		// Add highlighted match
		matched := text[actualIdx : actualIdx+len(m.searchTerm)]
		result.WriteString(highlightStyle.Render(matched))

		lastIdx = actualIdx + len(m.searchTerm)
	}

	return result.String()
}

// View renders the UI
func (m Model) View() string {
	if !m.ready && !m.inputMode {
		return "Searching...\n"
	}

	if m.quitting {
		return ""
	}

	// Input mode view
	if m.inputMode {
		var help string
		if m.searchTerm == "" {
			help = helpStyle.Render("enter: search • esc/ctrl+c: quit")
		} else {
			help = helpStyle.Render("enter: search • esc: cancel")
		}

		return fmt.Sprintf("\n🔍 Search diary entries:\n\n%s\n\n%s",
			m.searchInput.View(),
			help,
		)
	}

	// Results mode view
	help := helpStyle.Render("↑/↓: navigate • enter: view full entry • /: new search • q: quit")

	return fmt.Sprintf("%s\n\n%s\n%s",
		m.list.View(),
		m.viewport.View(),
		help,
	)
}

// GetOpenDate returns the date to open (if user pressed Enter)
func (m Model) GetOpenDate() string {
	return m.openDate
}

// GetSearchTerm returns the current search term
func (m Model) GetSearchTerm() string {
	return m.searchTerm
}

// GetSelectedIndex returns the currently selected list index
func (m Model) GetSelectedIndex() int {
	return m.list.Index()
}

// SetSelectedIndex sets the initial list selection index
// This is applied after search results are loaded
func (m *Model) SetSelectedIndex(index int) {
	m.initialIndex = index
}
