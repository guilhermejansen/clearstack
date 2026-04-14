# Development

## Prerequisites

- Go 1.24+
- golangci-lint v1.64+
- make
- `git`
- Optional: `goreleaser` (release-only), Docker (Docker detector smoke tests)

## Layout

```
cmd/clearstack/       # Cobra entry point, flags, per-command glue
internal/
  engine/             # Scanner, Sizer, Classifier, Cleaner, Safety, DormancyPolicy
  detectors/          # Detector interface + all registered categories
  trash/              # Portable recycle-bin (darwin / linux / windows / fallback)
  journal/            # Append-only JSONL operations log
  platform/           # OS detection + path helpers
  config/             # Viper-backed schema, defaults, profiles
  ui/tui/             # Bubble Tea interactive model/views
  version/            # Build-time version injected via ldflags
.github/workflows/    # CI matrix + release pipeline
```

## Typical workflow

```sh
make tidy              # go mod tidy
make fmt               # gofmt + goimports
make vet               # go vet ./...
make lint              # golangci-lint run
make test              # go test -race -count=1 ./...
make check             # all of the above in order
make build             # ./bin/clearstack
make cross             # build darwin/linux/windows × amd64/arm64
make snapshot          # goreleaser snapshot build
```

## Adding a new category

1. Decide whether the category matches by exact directory name (most common).
2. Register it from an `init()` in `internal/detectors/<stack>.go`:

```go
func init() {
    Register(&SimpleDirDetector{
        CategoryID:    "my_cache",
        DirName:       ".my-cache",
        Desc:          "My thing cache",
        SafetyLevel:   SafetySafe,
        Strategy:      StrategyTrash,
        NeedsDormancy: false,
        Require:       []string{"my.config"},
    })
}
```

3. For singleton caches (Go, pnpm, pip, ...) implement `Detector` directly
   and emit a single Match whose Path is the resolved cache location.
4. For external-command strategies implement the optional `NativeCommander`
   interface; the `Cleaner` will exec the returned argv.
5. Add a unit test — fixture-based is easiest; `internal/engine/scanner_test.go`
   is a good template.
6. Update `docs/CATEGORIES.md`.

## Safety rules

- Never follow symlinks (fastwalk `Follow=false` is enforced).
- New whitelist paths go into `internal/engine/safety.go::defaultProtected`.
- If a category can cause data loss, register it with `SafetyDanger`; the
  cleaner will require `--yes` + `--categories=<id>` explicitly.

## Running the TUI

```sh
go run ./cmd/clearstack ~/Developer
```

Use `tea.WithAltScreen()` and forgive the scanner if it takes a few seconds
the first time on a large tree — fastwalk is still faster than anything
else Go offers, but 200 GB is 200 GB.

## Releasing

Tag `vX.Y.Z` on `main`, push the tag, and the `release.yml` workflow does
the rest via GoReleaser.
