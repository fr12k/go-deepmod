# sync-replaces

Syncs `replace` directives from direct dependencies to your module's `go.mod`. Local path replaces are ignored.

## Install

```bash
go install github.com/fr12k/go-deepmod/cmd/sync-replaces@latest
```

## Usage

```bash
sync-replaces              # sync replaces
sync-replaces -n           # dry-run
sync-replaces -v           # verbose
sync-replaces -skip-tidy   # skip go mod tidy
sync-replaces -dir <path>  # specify module directory
```

## Integration

**go:generate (recommended)**
```go
//go:generate go run github.com/fr12k/go-deepmod/cmd/sync-replaces@latest
```

**Go 1.24+ tool directive**
```go
// go.mod
tool github.com/fr12k/go-deepmod/cmd/sync-replaces

// any .go file
//go:generate go tool sync-replaces
```

## How It Works

1. Lists all direct dependencies via `go list -m -json all`
2. Parses each dependency's `go.mod` for `replace` directives
3. Filters out local path replaces (`./`, `../`, `/absolute`)
4. Deduplicates conflicts (lowest version wins)
5. Applies replaces via `go mod edit -replace=...`
6. Adds source tracking comment: `// from: dep@version`
7. Removes stale replaces (tracked via `// from:` comments)

## Source Tracking

Each synced replace includes a comment tracking its origin:

```
replace github.com/foo => github.com/bar v1.0.0 // from: example.com/dep
```

When a dependency removes its replace directive, `sync-replaces` detects and removes the stale entry.

## Project Structure

```
cmd/sync-replaces/main.go     # CLI entry point
internal/
  pathutil/pathutil.go        # Local path detection
  gomod/gomod.go              # go.mod parsing, replace handling
  sync/sync.go                # Main sync orchestration
```

## License

MIT
