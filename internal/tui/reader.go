package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/storage"
)

type readerRenderedMsg struct {
	content string
}

type readerEntryLoadedMsg struct {
	date      string
	plaintext string
	err       error
}

// ReaderModel is an interactive markdown viewer with entry navigation
type ReaderModel struct {
	user         string
	keyPath      string
	entries      []string // all dates, sorted descending (newest first)
	currentIndex int
	date         string
	markdown     string // raw markdown for re-rendering on resize
	stylePath    string // pre-detected "dark" or "light"
	viewport     viewport.Model
	width        int
	ready        bool
	quitting     bool
}

// NewReaderModel creates a new reader TUI for viewing diary entries.
// Call this before tea.NewProgram so style detection happens outside alt-screen.
func NewReaderModel(user, keyPath string, entries []string, startIndex int) ReaderModel {
	style := "dark"
	if !termenv.HasDarkBackground() {
		style = "light"
	}
	return ReaderModel{
		user:         user,
		keyPath:      keyPath,
		entries:      entries,
		currentIndex: startIndex,
		date:         entries[startIndex],
		stylePath:    style,
	}
}

func (m ReaderModel) Init() tea.Cmd {
	return m.loadEntry(m.date)
}

func (m ReaderModel) loadEntry(date string) tea.Cmd {
	keyPath := m.keyPath
	user := m.user
	return func() tea.Msg {
		entryPath, err := storage.DiaryPath(user, date)
		if err != nil {
			return readerEntryLoadedMsg{date: date, err: err}
		}

		encrypted, err := os.ReadFile(entryPath)
		if err != nil {
			return readerEntryLoadedMsg{date: date, err: err}
		}

		plaintext, err := crypto.Decrypt(encrypted, keyPath)
		if err != nil {
			return readerEntryLoadedMsg{date: date, err: err}
		}

		return readerEntryLoadedMsg{date: date, plaintext: string(plaintext)}
	}
}

func (m ReaderModel) renderAsync() tea.Cmd {
	width := m.width - 6
	markdown := m.markdown
	stylePath := m.stylePath
	return func() tea.Msg {
		if width <= 0 {
			width = 80
		}
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStylePath(stylePath),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return readerRenderedMsg{content: markdown}
		}
		rendered, err := renderer.Render(markdown)
		if err != nil {
			return readerRenderedMsg{content: markdown}
		}
		return readerRenderedMsg{content: rendered}
	}
}

func (m ReaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-4)
			m.viewport.Style = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 4
		}
		if m.markdown != "" {
			return m, m.renderAsync()
		}

	case readerEntryLoadedMsg:
		if msg.err != nil {
			m.viewport.SetContent(fmt.Sprintf("Error loading entry: %v", msg.err))
			return m, nil
		}
		m.date = msg.date
		m.markdown = msg.plaintext
		m.viewport.GotoTop()
		return m, m.renderAsync()

	case readerRenderedMsg:
		m.viewport.SetContent(msg.content)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "J": // Shift+J: next (older) entry
			if m.currentIndex < len(m.entries)-1 {
				m.currentIndex++
				m.date = m.entries[m.currentIndex]
				m.markdown = ""
				m.viewport.SetContent("Loading...")
				return m, m.loadEntry(m.date)
			}

		case "K": // Shift+K: previous (newer) entry
			if m.currentIndex > 0 {
				m.currentIndex--
				m.date = m.entries[m.currentIndex]
				m.markdown = ""
				m.viewport.SetContent("Loading...")
				return m, m.loadEntry(m.date)
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ReaderModel) title() string {
	pos := fmt.Sprintf("[%d/%d]", m.currentIndex+1, len(m.entries))
	return fmt.Sprintf("📖 %s — %s  %s", m.user, m.date, pos)
}

func (m ReaderModel) View() string {
	if !m.ready {
		return "Loading...\n"
	}
	if m.quitting {
		return ""
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginLeft(2).
		Render(m.title())

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("  ↑/↓: scroll • J/K: prev/next entry • q/esc: quit")

	return fmt.Sprintf("%s\n%s\n%s", title, m.viewport.View(), help)
}
