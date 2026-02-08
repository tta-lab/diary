# Diary CLI - Final Design

## Command Structure

```bash
diary {user} {command} [args...]
```

## Commands

### `diary {user} append "text"`
**For agents** - Programmatic append to diary

- Appends text to today's diary entry
- Auto-creates new file if:
  - New day (no entry for today exists)
  - User's first diary entry ever
- Auto-appends to existing file if today's entry exists
- Encrypts and auto-commits after append

**Example:**
```bash
diary yuki append "Completed task #107 - diary encryption design"
diary neil append "Meeting with team about Q2 planning"
```

### `diary {user} edit [date]`
**For humans** - Interactive editing with $EDITOR

- Opens $EDITOR with decrypted content
- If no date: edits today's entry
- If date specified: edits that date's entry
- Creates new entry if doesn't exist
- Encrypts and auto-commits on save

**Example:**
```bash
diary neil edit              # Edit today
diary neil edit 2026-02-07   # Edit specific date
```

### `diary {user} read [date]`
**For reading** - Display diary entry

- If no date: shows **latest** diary entry for user
- If date specified: shows that specific entry
- Displays in terminal (plain text or interactive viewer with -t flag)

**Example:**
```bash
diary neil read              # Show latest entry
diary neil read 2026-02-07   # Show specific date
```

### `diary {user} list`
**For discovery** - Show all available dates

- Lists all diary entries for user
- Shows dates in descending order (newest first)
- Shows entry count

**Example:**
```bash
diary neil list
# Output:
# 2026-02-07
# 2026-02-06
# 2026-02-05
# ...
# Total: 42 entries
```

### `diary {user} search "term"`
**For searching** - Find across all entries

- Decrypts all entries (in memory)
- Searches for term
- Shows matching entries with context

**Example:**
```bash
diary neil search "task #107"
# Output:
# 2026-02-07:
#   Completed task #107 - diary encryption design
#   ...
```

## Storage Structure

```
~/.diary/
├── yuki/
│   ├── 2026-02-07.md.age
│   ├── 2026-02-06.md.age
│   └── 2026-02-05.md.age
└── neil/
    ├── 2026-02-07.md.age
    ├── 2026-02-06.md.age
    └── 2026-02-05.md.age
```

## Key Management

```
~/.config/diary/
├── yuki.age.key
└── neil.age.key
```

## Auto-Detection Logic for Append

```go
func handleAppend(user, text string) error {
    today := time.Now().Format("2006-01-02")
    entryPath := filepath.Join(os.Getenv("HOME"), ".diary", user, today + ".md.age")

    var content string

    // Check if today's entry exists
    if fileExists(entryPath) {
        // Decrypt existing content
        encrypted, _ := os.ReadFile(entryPath)
        content, _ = crypto.Decrypt(encrypted, keyPath)

        // Append new text
        content += "\n" + text
    } else {
        // New entry for today
        content = text
    }

    // Encrypt and save
    encrypted, _ := crypto.Encrypt([]byte(content), recipient)
    os.WriteFile(entryPath, encrypted, 0600)

    // Auto-commit
    git.AutoCommit(user, today)

    return nil
}
```

## Auto-Detection Logic for Read (Latest)

```go
func handleRead(user string, date string) error {
    if date == "" {
        // Find latest entry
        entries, _ := storage.ListEntries(user)
        if len(entries) == 0 {
            return fmt.Errorf("no entries found for user %s", user)
        }

        // Sort descending and take first (latest)
        sort.Slice(entries, func(i, j int) bool {
            return entries[i] > entries[j]
        })
        date = entries[0]
    }

    // Read and decrypt entry
    entryPath, _ := storage.DiaryPath(user, date)
    encrypted, _ := os.ReadFile(entryPath)
    plaintext, _ := crypto.Decrypt(encrypted, keyPath)

    // Display
    fmt.Println(string(plaintext))

    return nil
}
```

## Workflow Examples

### Agent appending to diary (programmatic)

```bash
# Morning - first entry of the day
diary yuki append "Started working on task #107"

# Later same day - auto-appends to existing entry
diary yuki append "Completed Phase 1 design"

# Result in 2026-02-07.md:
# Started working on task #107
# Completed Phase 1 design
```

### Human editing diary (interactive)

```bash
# Open today's diary in editor
diary neil edit

# Opens $EDITOR with decrypted content
# Human can add formatted markdown, restructure, etc.
# On save: encrypts and commits
```

### Reading latest diary

```bash
# Show my latest entry
diary neil read

# Output: (displays latest entry content)
```

### Discovering entries

```bash
# Show all available dates
diary neil list

# Output:
# 2026-02-07
# 2026-02-06
# 2026-02-05
# ...
```

## Multi-User Isolation

- Each user has separate encryption key
- Each user has separate storage directory
- No cross-user access without key sharing
- Git commits include user namespace

## Security

1. **Keys:** `~/.config/diary/{user}.age.key` (mode 0600)
2. **Entries:** `~/.diary/{user}/*.md.age` (encrypted at rest)
3. **Plaintext:** Only in RAM during read/edit/append
4. **Git:** Tracks encrypted files only

## Interactive Viewer

- `diary {user}` = latest entry in interactive viewer with J/K navigation
- `diary {user} read` = plain text output (for agents/scripts)
- `diary {user} read -t` = interactive viewer
- `diary {user} search` = interactive search TUI with detail view
