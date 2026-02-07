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

### 1. Generate encryption key

```bash
# Install age if not already installed
# macOS: brew install age
# Linux: apt install age / pacman -S age

# Generate key for your user
mkdir -p ~/.config/diary
age-keygen -o ~/.config/diary/${USER}.age.key

# IMPORTANT: Back up this key! If lost, diaries cannot be recovered
```

### 2. Initialize diary storage

```bash
mkdir -p ~/.diary/${USER}

# Optional: Initialize as git repository for version control
cd ~/.diary
git init
echo "*.key" > .gitignore
git add .gitignore
git commit -m "chore(diary): initialize diary repository"
```

## Usage

### Write diary entry

```bash
# Append to today's diary
diary append $(date +%Y-%m-%d)

# Append to specific date
diary append 2026-02-07
```

Your `$EDITOR` will open with the decrypted content (if entry exists) or blank file (if new).
Save and exit - the entry will be encrypted and auto-committed.

### Read diary entry

```bash
# Read today's diary
diary read $(date +%Y-%m-%d)

# Read specific date
diary read 2026-02-07
```

### List entries

```bash
# List all entries
diary list

# List with filter (TODO)
diary list 2026-02
```

### Search

```bash
# Search across all entries
diary search "keyword"
```

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

**Current**: v0.1.0 - Initial structure, commands stubbed

**TODO**:
- [ ] Implement age encryption/decryption
- [ ] Implement read command
- [ ] Implement append command with editor
- [ ] Implement list command
- [ ] Implement search command
- [ ] Git auto-commit integration
- [ ] Configuration file support
- [ ] Markdown rendering
- [ ] Error handling
- [ ] Tests
- [ ] Documentation

## License

MIT

## Author

Neil Guion
