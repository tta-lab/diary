---
name: diary
description: Encrypted diary for agents. Read, write, append, and search diary entries encrypted at rest with age. Use when you need to record reflections, log daily events, recall past entries, or search diary history.
---

# diary-cli — Encrypted Diary for Agents

Local-first, git-native, multi-user encrypted diary. Every entry is age-encrypted at rest, auto-committed to git, and isolated per user.

## Quick Start

```bash
# Write today's diary (appends to existing entry or creates new)
diary neil append "Had a productive morning. Shipped the auth refactor."

# Read a specific date as plain text
diary neil read 2026-02-17

# Search all entries
diary neil search "product launch"

# List all entry dates
diary neil list
```

## Commands

### append — Add to Today's Entry

```bash
diary <user> append "Your text here"
```

- Creates today's entry if it doesn't exist
- Appends to existing entry with a blank line separator
- Encrypts and auto-commits to git

### read — Read Entries

```bash
# Specific date, plain text to stdout
diary <user> read 2026-02-17

# Today's entry
diary <user> read
```

- Output is plain text to stdout — pipe-friendly

### search — Find Entries

```bash
diary <user> search "keyword"
```

- Decrypts and searches all entries in parallel
- Case-insensitive line matching
- Results sorted newest first

### list — List All Dates

```bash
diary <user> list
```

- One date per line, newest first
- Use to discover which dates have entries

### import — Bulk Import

```bash
diary <user> import /path/to/markdown/files
```

- Imports `YYYY-MM-DD.md` files from a directory
- Encrypts each and commits

## Agent Workflow

When writing a diary entry:

1. **Identify the user** — use the correct username (matches their diary namespace)
2. **Use `append`** — the only write command available to agents
3. **Write in first person** from the user's perspective, or third person if logging on behalf of a system
4. **One thought per append** is fine — multiple appends in a day accumulate naturally
5. **Include dates, names, and specifics** — vague entries are useless later

When reading diary entries:

1. **Search before browsing** — `diary <user> search "term"` is faster than reading everything
2. **Use `list`** to discover which dates have entries
3. **Use `read`** with a specific date to get full content

## Writing Good Diary Entries

A good diary entry is something the user or a future agent can actually use.

**Do:**
- Record what happened, what was decided, and why
- Include specifics: names, numbers, links, commit hashes
- Note emotional context — it helps recall and reflection
- Separate topics with markdown headers or blank lines

**Don't:**
- Write generic summaries ("had a good day")
- Duplicate information already in git commits or issue trackers
- Over-format — plain prose with occasional markdown is fine
- Include secrets, tokens, or passwords (entries are encrypted but git history is forever)

**Example entry:**

```markdown
## Morning

Paired with Sam on the billing migration. We decided to keep the old
table around for 30 days as a rollback safety net. PR #142.

## Afternoon

Product review went well. CEO wants to ship the dashboard by March 1.
I think that's tight but doable if we skip the export feature for now.

Feeling good about the team momentum this week.
```

## Storage and Configuration

| Path | Purpose |
|---|---|
| `~/.diary/<user>/YYYY-MM-DD.md.age` | Encrypted entries |
| `~/.config/diary/<user>.age.key` | User's age private key (mode 0600) |
| `~/.diary/.git/` | Auto-committed git repo |
| `~/.diary/logs/diary.log` | Debug log |

- First run auto-creates keys, directories, and git repo
- `.gitignore` prevents committing plaintext `.md` or `.age.key` files
- Each user has isolated keys and storage — no cross-user access

## Notes

- **Keys are not backed up automatically.** If you lose `~/.config/diary/<user>.age.key`, those entries are gone.
- **Git history contains only ciphertext.** Safe to push to a private remote for backup.
- **All encryption/decryption happens in RAM.** Plaintext never touches disk.
