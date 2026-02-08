# diary - Encrypted Diary CLI

A transparent encrypted diary management tool with multi-user support.

## Features

- **Encrypted at rest**: All diary entries encrypted with age
- **Transparent workflow**: Decrypt on read, encrypt on save
- **Multi-user**: Each user has separate encryption key
- **Auto-commit**: Automatic git commits on save
- **Search**: Full-text search across encrypted entries
- **Markdown**: Beautiful rendering with markdown support

## Installation

### From source

```bash
go install github.com/neilguion/diary-cli/cmd/diary@latest
```

### Manual build

```bash
git clone https://github.com/neilguion/diary-cli.git
cd diary-cli
go build -o diary ./cmd/diary
sudo mv diary /usr/local/bin/
```

## Setup

**Auto-setup on first use!** 🎉

When you run any diary command for the first time, the CLI automatically:
1. Generates an age encryption key for your user
2. Creates `~/.diary/${USER}/` directory
3. Initializes a git repository (if not exists)
4. Sets up `.gitignore` to exclude keys

Just start using it:

```bash
# First time use - everything auto-configured
diary neil edit

# That's it! Key generated, storage created, ready to go.
```

### Manual setup (optional)

If you prefer manual control:

```bash
# Install age if not already installed
# macOS: brew install age
# Linux: apt install age / pacman -S age

# Generate key for your user
mkdir -p ~/.config/diary
age-keygen -o ~/.config/diary/${USER}.age.key

# IMPORTANT: Back up this key! If lost, diaries cannot be recovered

# Create diary directory
mkdir -p ~/.diary/${USER}

# Optional: Initialize as git repository for version control
cd ~/.diary
git init
echo "*.key" > .gitignore
git add .gitignore
git commit -m "chore(diary): initialize diary repository"
```

## Usage

### Quick view (default)

```bash
# Read latest entry in interactive viewer (markdown rendering + scrolling)
diary neil

# Navigate between entries with J/K (shift+j/k)
# Same as: diary neil read -t
```

### Write diary entry

```bash
# Edit today's diary
diary neil edit

# Edit specific date
diary neil edit 2026-02-07

# Append text (for agents/scripts)
diary neil append "Completed task #107"
```

Your `$EDITOR` will open with the decrypted content (if entry exists) or blank file (if new).
Save and exit - the entry will be encrypted and auto-committed.

### Read diary entry

```bash
# Read latest entry (plain text)
diary neil read

# Read in interactive viewer (human-friendly)
diary neil read -t

# Read specific date
diary neil read 2026-02-07
```

### List entries

```bash
# List all entries (greppable output)
diary neil list

# Output: one date per line, newest first
# 2026-02-07
# 2026-02-05
# 2026-02-01
```

### Search (Interactive TUI)

```bash
# Search across all entries
diary neil search "encryption"
```

**Interactive TUI features:**
- 📋 Top pane: List of matching entries with match counts
- 👁️ Bottom pane: Live preview with highlighted search terms
- ⌨️ Navigation: Arrow keys to select, Enter to view full entry
- 🚪 Quit: Press 'q' or Esc

**Performance:**
- Parallel decryption for fast search
- Streams results as they're found
- Works efficiently with hundreds of entries

## Storage Structure

```
~/.diary/
├── ${USER}/
│   ├── 2026-02-07.md.age
│   ├── 2026-02-08.md.age
│   └── 2026-02-09.md.age
└── .git/
```

## Configuration

Optional configuration file: `~/.config/diary/config.toml`

```toml
[storage]
base_path = "~/.diary"

[user]
default_user = "neil"

[git]
auto_commit = true
commit_message_template = "feat(diary): update entry for {date}"

[editor]
command = "vim"  # Defaults to $EDITOR
```

## Security

- **Encryption**: age (modern, audited, simple)
- **Keys**: Stored locally in `~/.config/diary/${USER}.age.key`
- **Never on disk**: Decrypted content only in RAM during edit
- **Per-user isolation**: Each user has separate key and namespace

### Key backup

**CRITICAL**: Back up your age key! Without it, encrypted diaries cannot be recovered.

Recommended backup locations:
- Password manager (1Password, Bitwarden, etc.)
- Encrypted USB drive
- Printed on paper in safe location

## Multi-user

Multiple users on same machine can each have their own encrypted diaries:

```bash
# User 1
USER=alice age-keygen -o ~/.config/diary/alice.age.key
diary append $(date +%Y-%m-%d)  # Encrypts with alice's key

# User 2
USER=bob age-keygen -o ~/.config/diary/bob.age.key
diary append $(date +%Y-%m-%d)  # Encrypts with bob's key
```

Each user's diaries are isolated and cannot be read by others without the key.

## Development Status

**Current**: v0.2.0 - Core features complete

**Implemented** ✅:
- [x] Age encryption/decryption with armor encoding
- [x] Read command (plain + interactive markdown viewer)
- [x] Append command (for agents/scripts)
- [x] Edit command (opens $EDITOR, works with all editors)
- [x] List command (greppable output)
- [x] Search command (interactive TUI with bubbletea)
- [x] Git auto-commit integration
- [x] Multi-user support with auto-setup
- [x] Markdown rendering with Glamour (in-app, no external dependency)
- [x] Error handling
- [x] Tests for core encryption
- [x] Basic documentation

**TODO**:
- [ ] Configuration file support (config.toml)
- [ ] List with date filter (e.g., `list 2026-02`)
- [ ] Export functionality
- [ ] Search results export
- [ ] Comprehensive test suite
- [ ] CI/CD pipeline

## License

MIT

## Author

Neil Guion
