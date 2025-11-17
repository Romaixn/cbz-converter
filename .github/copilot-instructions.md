## Review Philosophy
- Only comment when you have HIGH CONFIDENCE (>80%) that an issue exists.
- Be concise: one sentence per comment when possible.
- Focus on actionable feedback, not observations.
- When reviewing text (comments, docs), only comment on clarity issues if the text is genuinely confusing or could lead to errors.

## Priority Areas (Review These First)

### 🚨 Security & Safety
- Use of the `unsafe` package without clear justification.
- Command injection risks (via `os/exec` with user input).
- Path traversal vulnerabilities (file path manipulation).
- Credential exposure or hardcoded secrets.
- Missing input validation on external data (e.g., HTTP requests, files).
- Improper error handling that could leak sensitive info.

### 🐛 Correctness & Bugs
- Logic errors that could cause panics or incorrect behavior.
- **Race conditions** in goroutines (concurrent access to maps, slices, or variables without a mutex).
- Resource leaks (unclosed files, network connections, tickers; **look for missing `defer close()`**).
- Off-by-one errors or boundary conditions.
- **Improper Error Handling**:
    - Errors being ignored (assigned to `_`).
    - Using `panic()` where returning an `error` is appropriate.
    - Not using `fmt.Errorf` with `%w` to wrap errors and preserve context.
- **Unnecessary Pointers**: Using pointers (e.g., `*string`, `*int`) where a value type is sufficient (adding nil-check complexity).
- Unnecessary `*bool` (a simple `bool` already defaults to `false`).
- Overly defensive code that adds unnecessary `nil` checks.
- Unnecessary comments that just restate what the code already shows (suggest removing them).

### 🏛️ Architecture & Patterns
- Code that violates existing patterns in the codebase.
- Functions that should return an `error` but do not.
- **Goroutine Misuse**:
    - Starting a "fire-and-forget" goroutine without a `sync.WaitGroup` or channel for synchronization.
    - Blocking operations in critical contexts.
- Improper interface implementations (e.g., confusion between pointer and value receivers).

---

## Project-Specific Context

- This is a Go project using **Go Modules** (`go.mod`).
- Core packages: `cbz-converter` (CLI)
- Error handling: Use the standard `error` interface. Use `fmt.Errorf("context: %w", err)` to wrap errors.
- Concurrency: Goroutines and Channels (Go standard).

---

## ⚠️ Basic Checks (Since CI is Missing)

**Important**: Because no CI pipeline is configured, you **must** also flag the following basic issues that would normally be automated.

- **Formatting**: Code that is visibly not formatted with `gofmt` or `goimports`.
- **Basic Linting**: Obvious issues that `go vet` or `golangci-lint` would catch:
    - Variables declared but not used.
    - Unused imports.
    - Unnecessary assignments.
- **Dependencies**: Imports of packages that do not appear to be listed in the `go.mod`.
- **Tests**: Exported public functions lacking associated unit tests, or changes that seem to invalidate existing tests.
- **Syntax**: Obvious syntax or compilation errors.

---

## Skip These (Low Value)

Do not comment on:
- **Minor naming suggestions** (unless truly confusing).
- **Suggestions to add comments** (for self-documenting code).
- **Refactoring suggestions** (unless there’s a clear bug or maintainability issue).
- **Multiple issues in one comment** (choose the single most critical issue).
- **Logging suggestions** (unless for errors or security events; the codebase might need *less* logging, not more).
- **Pedantic accuracy in text** (unless it would cause actual confusion).

---

## Response Format

When you identify an issue:
1.  **State the problem** (1 sentence)
2.  **Why it matters** (1 sentence, only if not obvious)
3.  **Suggested fix** (code snippet or specific action)

**Example:**
> This function ignores the error returned by `file.Close()`.
> If the close fails (e.g., a buffered write couldn't be flushed), data could be silently corrupted.
> Suggestion:
> ```go
> defer func() {
>     if err := file.Close(); err != nil {
>         log.Printf("Error closing file: %v", err)
>     }
> }()
> ```

## When to Stay Silent
If you’re uncertain whether something is an issue, don’t comment. False positives create noise and reduce trust in the review process.
