# Search TUI Documentation

## Overview

The diary search feature provides an interactive Terminal User Interface (TUI) for searching across all encrypted diary entries with real-time preview and highlighting.

## Architecture

### Components

```
┌─────────────────────────────────────────────┐
│  Search: "encryption" (2 entries)           │
│                                             │
│  > 2026-02-07 (3 matches)                  │  ← Top Pane: List
│    2026-02-05 (1 match)                    │    (bubbles/list)
│                                             │
└─────────────────────────────────────────────┘
┌─────────────────────────────────────────────┐
│  📅 2026-02-07                              │
│                                             │
│  Line 10:                                   │  ← Bottom Pane: Preview
│  Started work on encryption feature         │    (bubbles/viewport)
│                     ^^^^^^^^^^              │    with highlighting
│                                             │
│  Press Enter to view full entry             │
└─────────────────────────────────────────────┘

  ↑/↓: navigate • enter: view full entry • q: quit
```

### View Modes

The search TUI has three modes:

1. **Input Mode** — user types a search term (shown when no term provided)
2. **Search View** — split pane with results list and preview
3. **Detail View** — full-screen Glamour-rendered markdown viewer with scrolling

### Tech Stack

- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) — Elm-inspired TUI framework
- **Markdown Rendering**: [Glamour](https://github.com/charmbracelet/glamour) — renders markdown to ANSI-styled strings
- **Components**:
  - `bubbles/list` — selectable list of search results
  - `bubbles/viewport` — scrollable preview and detail panes
  - `bubbles/textinput` — search term input
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling library

### Data Flow

```
User enters: diary neil search "encryption"
                    ↓
            Launch TUI (bubbletea)
                    ↓
        Perform parallel search
        (decrypt all entries concurrently)
                    ↓
            Build result list
        (date + match count + preview)
                    ↓
        Render split-pane TUI
    (list on top, preview on bottom)
                    ↓
    User navigates with arrow keys
    (preview updates for selected item)
                    ↓
      User presses Enter
                    ↓
    Load and decrypt selected entry
                    ↓
    Render with Glamour (async)
                    ↓
    Display in detail viewport
    (scrollable, full-screen)
```

## Detail View: Glamour Rendering

When the user presses Enter on a search result, the entry is loaded in two async phases:

1. **Decrypt** (`loadEntry`) — reads and decrypts the `.md.age` file
2. **Render** (`renderContent`) — renders markdown with Glamour using pre-detected style

```go
// Phase 1: Decrypt (async)
func (m Model) loadEntry(date string) tea.Cmd {
    return func() tea.Msg {
        plaintext, _ := crypto.Decrypt(encrypted, m.keyPath)
        return entryLoadedMsg{date: date, plaintext: string(plaintext)}
    }
}

// Phase 2: Render (async, uses pre-detected style)
func (m Model) renderContent() tea.Cmd {
    return func() tea.Msg {
        renderer, _ := glamour.NewTermRenderer(
            glamour.WithStylePath(m.glamourStyle), // "dark" or "light"
            glamour.WithWordWrap(width),
        )
        rendered, _ := renderer.Render(m.detailPlaintext)
        return contentRenderedMsg{content: rendered}
    }
}
```

Terminal style is pre-detected in `NewSearchModel()` before entering alt-screen to avoid the slow `termenv.HasDarkBackground()` query inside alt-screen mode. See [GLAMOUR-OPTIMIZATION.md](GLAMOUR-OPTIMIZATION.md) for details.

## Performance Optimizations

### Parallel Decryption

```go
// Search in parallel using goroutines
resultChan := make(chan SearchResult, len(entries))
var wg sync.WaitGroup

for _, date := range entries {
    wg.Add(1)
    go func(date string) {
        defer wg.Done()
        result := searchEntry(date)
        if len(result.matches) > 0 {
            resultChan <- result
        }
    }(date)
}
```

**Benefits:**
- Searches 100+ entries in ~200ms
- CPU-bound task scales with cores
- No sequential bottleneck

### Async Rendering

Glamour rendering runs via `tea.Cmd` to avoid blocking the UI. The viewport appears immediately while markdown renders in the background.

### Match Limiting

Preview pane shows first 5 matches per entry to avoid clutter. Additional matches indicated with "... and N more matches".

## UX Design

### Highlighting Strategy

Search terms are highlighted using terminal ANSI color codes:

```go
highlightStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("226")).  // Yellow
    Background(lipgloss.Color("235")).  // Dark gray
    Bold(true)
```

### Context Display

Each match shows:
1. **Line number** — position in the original entry
2. **Matched line** — full line containing the search term
3. **Highlighting** — search term emphasized with color

### Navigation

| Key | Action |
|-----|--------|
| `↑`/`k` | Previous entry |
| `↓`/`j` | Next entry |
| `Enter` | View full entry (Glamour-rendered detail view) |
| `/` | New search |
| `q`/`Esc`/`Ctrl+C` | Quit search |

**In detail view:**

| Key | Action |
|-----|--------|
| `↑`/`↓`/`PgUp`/`PgDn` | Scroll content |
| `q`/`Esc` | Back to search results |
| `Ctrl+C` | Quit |

### Workflow

```bash
# User flow:
diary neil search "encryption"
# → TUI opens, shows results
# → User selects 2026-02-07
# → Presses Enter
# → Detail view opens with Glamour-rendered markdown
# → User scrolls through entry
# → Presses q/esc to return to search results
# → Selection is preserved
```

## Implementation Details

### Model Structure

```go
type Model struct {
    user           string
    searchTerm     string
    results        []SearchResult
    list           list.Model         // Top pane
    viewport       viewport.Model     // Bottom pane (preview)
    searchInput    textinput.Model    // Search input field
    inputMode      bool               // True when typing search term
    glamourStyle   string             // Pre-detected "dark" or "light"

    // Detail view state
    mode              viewMode        // searchView or detailView
    detailViewport    viewport.Model  // Full-screen detail viewport
    detailDate        string
    detailPlaintext   string          // Raw markdown for re-rendering on resize
}
```

### Update Loop

Bubbletea uses Elm architecture:

```
View → User Input → Update → View → ...
```

1. **Init**: Launch parallel search (or enter input mode)
2. **Update**: Handle messages (search complete, entry loaded, content rendered, key press, window resize)
3. **View**: Render based on current mode (input / search results / detail)

### Message Types

```go
searchCompleteMsg  // Search finished, populate list
entryLoadedMsg     // Entry decrypted, switch to detail view
contentRenderedMsg // Glamour render complete, set viewport content
```

## Security Considerations

### Memory Safety

- Decrypted content only in RAM
- No temporary files created
- Content wiped when TUI exits
- Goroutines clean up automatically

### Key Access

Search requires user's age key:
```go
keyPath, err := crypto.KeyPath(user)
// ~/.config/diary/{user}.age.key
```

Only user with valid key can search their entries.

## Testing

### Automated Testing

- Unit tests for search logic (match finding, highlighting, context extraction)
- Integration tests for full workflow (encrypt → search → verify matches)

### Manual Testing

```bash
diary neil search "term"     # Interactive search
diary neil search            # Opens input mode first
```

## References

- [Bubbletea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Glamour](https://github.com/charmbracelet/glamour)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [GLAMOUR-OPTIMIZATION.md](GLAMOUR-OPTIMIZATION.md) — detailed notes on the Glamour integration
