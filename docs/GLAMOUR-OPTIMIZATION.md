# Glamour Optimization: Replacing Glow with Direct Rendering

## Background

diary-cli originally used **Glow** (the Charmbracelet CLI tool) as an external process for markdown rendering. Since diary-cli already depends on the Charmbracelet stack (Bubbletea, Bubbles, Lipgloss), Glow was redundant — it's just a wrapper around Glamour + Bubbles Viewport.

### Before (Glow CLI)

```
diary-cli → exec.Command("glow", "-p", file) → subprocess renders markdown
```

- Required Glow binary installed on the system
- Spawned a subprocess for each render
- Used temp files or stdin piping
- Fallback to plain text if Glow not installed

### After (Direct Glamour + Viewport)

```
diary-cli → glamour.Render() → viewport.SetContent() → in-process TUI
```

- No external dependencies
- In-process rendering
- Consistent TUI experience across all views
- Full control over rendering and viewport behavior

## Architecture

### Two rendering contexts

1. **Reader TUI** (`internal/tui/reader.go`) — standalone interactive viewer for `diary {user}` and `diary {user} read -t`
2. **Search TUI detail view** (`internal/tui/search.go`) — entry viewer within the search interface

Both follow the same pattern:

```
Decrypt → Store plaintext → WindowSizeMsg → Async Glamour render → Set viewport content
```

### Reader TUI

```go
type ReaderModel struct {
    title     string
    markdown  string // raw markdown for re-rendering on resize
    stylePath string // pre-detected "dark" or "light"
    viewport  viewport.Model
    width     int
    ready     bool
    quitting  bool
}
```

Lifecycle:
1. `NewReaderModel()` — detects terminal style (dark/light) before alt-screen
2. `WindowSizeMsg` — sets up viewport dimensions, triggers async render
3. `readerRenderedMsg` — sets rendered content on viewport
4. User scrolls with arrow keys, quits with q/esc

### Search TUI detail view

Same pattern but embedded within the search model's `detailView` mode. When a user presses Enter on a search result:
1. `loadEntry()` — decrypts the entry (async)
2. `entryLoadedMsg` — stores plaintext, switches to detail view
3. `renderContent()` — renders with Glamour (async)
4. `contentRenderedMsg` — sets content on detail viewport

## Key Lessons

### 1. `glamour.WithAutoStyle()` is slow inside alt-screen

**Problem**: `glamour.WithAutoStyle()` calls `termenv.HasDarkBackground()` which sends OSC escape sequences to query the terminal's background color. Inside Bubbletea's alt-screen mode, this query can take **2-5 seconds** to time out.

**Fix**: Pre-detect the terminal style *before* creating `tea.NewProgram`:

```go
func NewReaderModel(title, markdown string) ReaderModel {
    // Detect BEFORE alt-screen
    style := "dark"
    if !termenv.HasDarkBackground() {
        style = "light"
    }
    return ReaderModel{
        stylePath: style,
        // ...
    }
}

// Later, inside alt-screen, use cached style:
renderer, _ := glamour.NewTermRenderer(
    glamour.WithStylePath(style), // not WithAutoStyle()
    glamour.WithWordWrap(width),
)
```

**Rule**: Any `termenv` terminal queries must happen before `tea.WithAltScreen()`. Once in alt-screen, terminal query responses are unreliable or slow.

### 2. Render asynchronously via tea.Cmd

**Problem**: Synchronous Glamour rendering inside `Update()` blocks the UI thread. Even without the auto-style issue, rendering large markdown blocks the viewport from appearing.

**Fix**: Return rendering as a `tea.Cmd` so it runs in a goroutine:

```go
func (m ReaderModel) renderAsync() tea.Cmd {
    width := m.width - 6
    markdown := m.markdown
    stylePath := m.stylePath
    return func() tea.Msg {
        renderer, _ := glamour.NewTermRenderer(
            glamour.WithStylePath(stylePath),
            glamour.WithWordWrap(width),
        )
        rendered, _ := renderer.Render(markdown)
        return readerRenderedMsg{content: rendered}
    }
}
```

**Important**: Capture values from the model *before* the closure. The closure runs in a separate goroutine and must not access the model directly.

### 3. Re-render on resize

When the terminal is resized (`WindowSizeMsg`), the markdown must be re-rendered with the new width. This is why we store the raw markdown (`m.markdown` / `m.detailPlaintext`) separately from the rendered output.

```go
case tea.WindowSizeMsg:
    m.viewport.Width = msg.Width - 4
    m.viewport.Height = msg.Height - 4
    return m, m.renderAsync() // re-render with new width
```

### 4. Two-phase loading in search detail view

The search TUI separates decryption from rendering:

```
Enter pressed → loadEntry() → entryLoadedMsg → renderContent() → contentRenderedMsg
                 (decrypt)                       (glamour)
```

This separation exists because:
- Decryption doesn't need terminal width
- Rendering needs viewport width (from `WindowSizeMsg`)
- Both are async operations that shouldn't block the UI

### 5. Guard against stale async messages

When the user navigates away before an async render completes, the message arrives in the wrong view mode. Guard against this:

```go
case contentRenderedMsg:
    if m.mode == detailView {
        m.detailViewport.SetContent(msg.content)
    }
    // else: ignore — user already left detail view
```

## Command behavior

| Command | Mode | Output |
|---------|------|--------|
| `diary neil` | Interactive TUI | Latest entry in scrollable viewer |
| `diary neil read` | Plain text | Decrypted markdown to stdout |
| `diary neil read -t` | Interactive TUI | Latest entry in scrollable viewer |
| `diary neil read 2026-02-07` | Plain text | Specific date to stdout |
| `diary neil read 2026-02-07 -t` | Interactive TUI | Specific date in scrollable viewer |
| `diary neil search "term"` | Interactive TUI | Search results with detail view |

## Dependencies

After optimization, the markdown rendering stack is:

```
github.com/charmbracelet/glamour     — Markdown → ANSI string
github.com/charmbracelet/bubbles     — Viewport (scrollable container)
github.com/charmbracelet/bubbletea   — TUI framework (Elm architecture)
github.com/charmbracelet/lipgloss    — Terminal styling
github.com/muesli/termenv            — Terminal capability detection
```

No external binaries required. Everything is compiled into the diary binary.

## Performance

| Operation | Before (Glow) | After (Glamour) |
|-----------|---------------|-----------------|
| First render | ~500ms (subprocess spawn) | ~50ms (in-process) |
| Subsequent renders | ~500ms (new subprocess) | ~20ms (no init overhead) |
| Style detection | N/A (Glow handles) | ~0ms (pre-cached) |
| Binary dependencies | Glow must be installed | None |

## Gotchas

1. **Never call `glamour.WithAutoStyle()` inside alt-screen** — it will hang for seconds waiting for terminal response
2. **Never call `termenv.HasDarkBackground()` inside alt-screen** — same issue, this is what `WithAutoStyle()` calls internally
3. **Always capture model values before closures in `tea.Cmd`** — closures run in goroutines; accessing the model directly is a race condition
4. **Always store raw markdown for re-rendering** — you need it when the terminal resizes
5. **Guard against stale async messages** — user may navigate away before render completes
6. **Use `glamour.WithWordWrap(width)`** — without this, long lines overflow the viewport border
