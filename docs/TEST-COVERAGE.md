# Test Coverage Report

**Last Updated:** 2026-02-07
**Version:** 0.2.0
**Total Tests:** 28 passing
**Test Time:** ~2 seconds

## Executive Summary

### Coverage by Package

| Package | Unit Tests | Integration | Coverage | Status |
|---------|-----------|-------------|----------|--------|
| `crypto` | 3 | Included | 69.5% | ✅ Good |
| `storage` | 6 | Included | 85.0% | ✅ Excellent |
| `setup` | 2 | Included | 79.7% | ✅ Good |
| `editor` | 4 | - | 14.7% | ⚠️ Low (by design) |
| `tui` | 8 | Included | 21.1% | ⚠️ Low (by design) |
| `git` | 0 | - | 0% | ⊘ Skipped |
| `cmd` | 0 | - | 0% | 📋 Future |
| **Integration** | - | **5** | - | ✅ Complete |

### Overall Status

- **28 tests** total, all passing
- **High coverage** for critical data paths (crypto, storage, setup)
- **Low coverage** for UI components (by design - hard to test)
- **No external dependencies** (git, editor, TUI skipped)
- **Fast execution** (~2 seconds)
- **CI-ready** (no flaky tests)

## Test Breakdown

### Phase 1: Unit Tests (23 tests)

#### Crypto Package (3 tests)
**File:** `internal/crypto/age_test.go`

```
✅ TestEncryptDecrypt
   - Basic roundtrip: plaintext → encrypt → decrypt → verify

✅ TestEncryptDecryptLargeText
   - Large content (10KB): verify performance and correctness

✅ TestDecryptWithWrongKey
   - Security: wrong key should fail decryption
```

**Coverage:** 69.5%
**What's tested:** Core encryption/decryption logic, key handling
**What's not tested:** Key generation (tested in setup package)

---

#### Storage Package (6 tests)
**File:** `internal/storage/filesystem_test.go`

```
✅ TestDiaryPath (5 sub-tests)
   - Valid dates: YYYY-MM-DD format
   - Invalid dates: wrong format, non-dates
   - Path structure verification

✅ TestListEntries
   - List all valid entries
   - Filter out invalid files (.txt, .md without .age)
   - Empty directory handling

✅ TestListEntriesNonexistentUser
   - Empty result for non-existent user (not error)

✅ TestEnsureDiaryDir
   - Directory creation with correct permissions (0700)
   - Idempotent (safe to call multiple times)

✅ TestReadEntry
   - Read encrypted file content
   - File path handling

✅ TestReadEntryNonexistent
   - Error handling for missing files
```

**Coverage:** 85.0%
**What's tested:** Path generation, file operations, directory management
**What's not tested:** Actual encryption/decryption (tested in crypto)

---

#### Setup Package (2 tests)
**File:** `internal/setup/setup_test.go`

```
✅ TestEnsureUserSetup
   - Key generation (age-keygen)
   - Directory creation (.config/diary, .diary)
   - Git repository initialization
   - Correct permissions and structure

✅ TestMultipleUsers
   - Multiple users in same environment
   - Separate keys for each user
   - Isolated directories
```

**Coverage:** 79.7%
**What's tested:** User setup workflow, multi-user support
**What's not tested:** Git configuration details (external dependency)

---

#### TUI Package (8 tests)
**File:** `internal/tui/search_test.go`

```
✅ TestHighlightTerm (5 sub-tests)
   - Simple match, case-insensitive
   - Multiple matches, partial words
   - No match (unchanged text)

✅ TestGetContext (4 sub-tests)
   - Middle of file (3 lines context)
   - First line (2 lines: match + after)
   - Last line (2 lines: before + match)
   - Single line (1 line: just match)

✅ TestSearchResultTitle
   - Format: "date (N matches)"

✅ TestSearchResultDescription (3 sub-tests)
   - With matches (preview first match)
   - No matches (empty string)
   - Long line truncation (60 chars + "...")

✅ TestSearchResultFilterValue
   - Returns date for filtering

✅ TestModelGetSearchTerm
   - Getter for search term

✅ TestModelGetOpenDate (2 sub-tests)
   - With date, empty date
```

**Coverage:** 21.1%
**What's tested:** Search logic helpers (highlight, context, formatting)
**What's not tested:** TUI rendering, bubbletea integration (UI framework handles this)

---

#### Editor Package (4 tests)
**File:** `internal/editor/open_test.go`

```
✅ TestGetEditor (6 sub-tests)
   - $EDITOR set
   - $VISUAL set (fallback)
   - EDITOR takes precedence
   - Default vim
   - Editor with args ("code --wait")
   - Helix editor (hx)

✅ TestGetEditorEnvPrecedence
   - EDITOR > VISUAL precedence

✅ TestGetEditorVisualFallback
   - VISUAL used when EDITOR empty

✅ TestGetEditorDefaultVim
   - vim default when neither set
```

**Coverage:** 14.7%
**What's tested:** Editor detection logic
**What's not tested:** Editor spawning, /dev/tty handling (requires real terminal)

---

### Phase 2a: Integration Tests (5 tests)

**File:** `internal/integration_test.go`

#### TestEncryptedWriteRead ✅
**Full workflow integration:**
```
Setup user → Generate key → Encrypt content →
Write to storage → List entries → Read entry →
Decrypt → Verify content matches
```

**Tests:**
- Complete data flow: crypto + storage
- Key loading and usage
- Directory structure
- File listing

---

#### TestMultiUserIsolation ✅
**Security and isolation:**
```
Setup 3 users (alice, bob, charlie) →
Each encrypts unique content →
Verify each can decrypt their own →
Verify alice CANNOT decrypt bob's entry (wrong key) →
Verify isolated directories
```

**Tests:**
- Multi-user security
- Key isolation
- Cross-user decryption prevention
- Separate storage namespaces

---

#### TestSearchRealEncryptedFiles ✅
**Search with real encrypted data:**
```
Setup user → Create 5 encrypted entries →
Search "encryption" → Verify 2 matches →
Search "bugs" → Verify 1 match →
Search "nonexistent" → Verify 0 matches →
Search case-insensitive → Verify works
```

**Sub-tests:**
- search_encryption
- search_bugs
- search_no_matches
- search_case_insensitive

**Tests:**
- Search + storage + crypto integration
- Case-insensitive matching
- Search accuracy

---

#### TestImportWorkflow ✅
**Import markdown files:**
```
Create 3 markdown files (2026-02-01.md, etc.) →
Create 2 invalid files (invalid.txt, no .md) →
Import directory → Encrypt valid files →
Verify 3 entries imported →
Verify content integrity →
Verify invalid files ignored
```

**Tests:**
- Import command workflow
- File filtering (*.md only)
- Content preservation
- Encryption after import

---

#### TestLargeDataset ✅
**Performance and scalability:**
```
Create 50 encrypted entries across 2 months →
List all entries → Verify 50 found →
Search across all → Verify all matches found
```

**Tests:**
- Performance with realistic dataset
- Search scalability
- List performance

**Note:** Skipped in short mode (`go test -short`)

---

## What's NOT Tested (By Design)

### 1. Git Commands
**Why:** External dependency, requires git installed and configured

**Skipped:**
- Auto-commit functionality
- Git repository validation
- Commit message formatting

**Alternatives:**
- Manual testing
- Optional tests (skip if git not available)

---

### 2. Editor Spawning
**Why:** Requires real $EDITOR, user interaction, /dev/tty

**Skipped:**
- Opening editor process
- Editor input/output
- /dev/tty handling

**What's tested instead:**
- Editor detection logic
- Argument parsing ("code --wait")
- Env var precedence

---

### 3. TUI Rendering
**Why:** Requires TTY, bubbletea framework already tested

**Skipped:**
- Bubbletea UI rendering
- Interactive navigation
- Key press handling
- Terminal output

**What's tested instead:**
- Search logic (highlight, context)
- Data structures (SearchResult)
- Helper functions

---

### 4. Glamour/Viewport Rendering
**Why:** Requires TTY for interactive viewer

**Skipped:**
- Glamour markdown rendering output
- Viewport scrolling interaction
- Terminal output

**What's tested instead:**
- Read command logic
- Content decryption
- Plain text output

---

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package
go test ./internal/crypto -v

# Run specific test
go test ./internal/storage -run TestDiaryPath -v

# Skip slow tests
go test ./... -short
```

### Coverage Reports

```bash
# Coverage summary
go test ./... -cover

# Detailed coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Coverage by function
go tool cover -func=coverage.out
```

### CI/CD Integration

```bash
# Fast test suite (no slow tests)
go test ./... -short -cover

# Full test suite
go test ./... -cover -timeout 30s
```

---

## Test Principles

### 1. Isolation
- Every test uses `t.TempDir()` for filesystem operations
- Every test uses `t.Setenv()` for environment variables
- No shared state between tests
- Automatic cleanup (temp directories removed)

### 2. No External Dependencies
- No git commands (skipped)
- No editor spawning (detection only)
- No TUI rendering (logic only)
- Self-contained tests

### 3. Fast Execution
- All tests run in ~2 seconds
- Parallel execution when safe
- No network calls
- No slow external processes

### 4. Deterministic
- No flaky tests
- No timing-dependent assertions
- No random data (predictable test data)
- Reproducible results

### 5. Table-Driven Tests
```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{...}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic
    })
}
```

---

## Future Test Plans

### Phase 2b: Command Handlers (Optional)

**Medium effort, medium value:**
```go
TestAppendCommand()
  - Mock git or skip auto-commit
  - Test full append workflow

TestReadCommand()
  - Test plain text mode (no TUI)
  - Plain text output verification

TestImportCommandHandler()
  - Test actual handleImport function
  - Stderr output capture
```

**Challenges:**
- Capturing stderr output
- Mocking git commands
- Setting os.Args for CLI

**ROI:** Medium - mostly glue code testing

---

### Phase 2c: End-to-End (Not Recommended)

**High effort, low value:**
```go
TestCompleteUserJourney()
  - Setup → Edit → Search → Read
  - Real editor, real git, real TUI
```

**Challenges:**
- Requires TTY simulation
- Requires git installed
- Requires editor installed
- High maintenance
- Fragile tests

**ROI:** Low - too much effort for marginal benefit

---

## Continuous Improvement

### Coverage Goals

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| crypto | 69.5% | 80% | Low (good enough) |
| storage | 85.0% | 90% | Low (excellent) |
| setup | 79.7% | 85% | Low (good enough) |
| editor | 14.7% | 20% | Low (by design) |
| tui | 21.1% | 30% | Medium (add more helpers) |
| git | 0% | 50% | Low (skip or mock) |
| cmd | 0% | 30% | Medium (command handlers) |

### Next Steps

1. **Add git tests with skip** (if git not available)
2. **Add more TUI helper tests** (search algorithms)
3. **Consider command handler tests** (if time permits)

---

## Maintenance

### When to Update This Document

- After adding new tests
- After major refactoring
- Before releases
- When coverage changes significantly (±10%)

### Test Ownership

- **Crypto tests:** Critical for security
- **Storage tests:** Critical for data integrity
- **Integration tests:** Critical for workflows
- **Other tests:** Important for regressions

### Test Failures

If tests fail:
1. Check test output for specific failure
2. Verify test environment (temp dirs, env vars)
3. Check if external dependency changed
4. Update test expectations if behavior changed intentionally

---

## Conclusion

### Strengths ✅

- **Excellent coverage** for critical paths (crypto, storage, setup)
- **Fast execution** (~2 seconds)
- **No flaky tests** (no external dependencies)
- **Well-organized** (unit + integration)
- **CI-ready** (works in any environment)

### Acceptable Gaps ⚠️

- **Low coverage** for UI (editor, TUI, Glamour rendering)
  - *Reason:* Hard to test, low ROI
- **No git tests**
  - *Reason:* External dependency
- **No command handler tests**
  - *Reason:* Phase 2b, optional

### Recommendations

1. **Keep current test suite** - excellent balance of coverage vs effort
2. **Don't add TUI rendering tests** - too complex, low value
3. **Consider git tests with skip** - if time permits
4. **Update this doc** - after adding new features

**Test suite status: Production-ready ✅**
