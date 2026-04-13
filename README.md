# clearstack

> Safe, cross-platform developer disk cleanup — like `npkill`, but for every stack.

`clearstack` is a single-binary Go tool that scans your dev machine for dormant build artifacts, caches, and package manager stores, and lets you reclaim disk space without breaking any active project.

**Safe-by-default**: dry-run first, trash-over-delete, whitelist protection, official commands (`pnpm store prune`, `go clean -modcache`, `docker system prune`) instead of raw `rm -rf` whenever possible.

## Why

Full-stack developers routinely sit on **tens to hundreds of gigabytes** of:

- `node_modules`, `.next`, `.turbo`, `.nuxt`, `.svelte-kit`, `.parcel-cache`, `.vite`
- `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, dormant `.venv`
- `GOCACHE`, `GOMODCACHE`
- Rust `target/`
- Gradle / Maven / NuGet local caches
- Xcode `DerivedData` (often 30–100 GB), old simulators, archives
- Docker dangling images, build cache, stopped containers
- npm / yarn / pnpm / bun global caches

Existing tools each cover one slice. `clearstack` covers them all with one interface.

## Highlights

- 🧹 **Multi-stack** — Node, Bun, Python, Go, Rust, Java, .NET, Docker, Xcode, and more.
- 🖥️ **Cross-platform** — macOS, Linux, Windows (single static binary, no CGO).
- 🎨 **Interactive TUI** — built on Charm's Bubble Tea, npkill-style navigation.
- 🧠 **Smart dormancy** — only touches projects idle for N days (configurable).
- 🛡️ **Safety net** — whitelist, dry-run default, trash-over-delete, operation journal, undo.
- ⚙️ **Scriptable CLI** — `scan`, `clean`, `analyze`, `doctor`, `undo`, `config`, JSON output.
- 🚀 **Fast** — parallel scan with `fastwalk`, lazy size calculation.

## Install

```sh
# Homebrew (macOS, Linux)
brew install guilhermejansen/tap/clearstack

# Scoop (Windows)
scoop bucket add guilhermejansen https://github.com/guilhermejansen/scoop-bucket
scoop install clearstack

# Direct (any OS)
curl -fsSL https://raw.githubusercontent.com/guilhermejansen/clearstack/main/install.sh | bash

# Go developers
go install github.com/guilhermejansen/clearstack/cmd/clearstack@latest
```

## Quick start

```sh
# Open the interactive TUI
clearstack

# Scan a directory without touching anything
clearstack scan ~/Developer --json

# Clean node_modules and .next in dormant projects (>14 days)
clearstack clean ~/Developer --categories=node_modules,.next --dry-run

# Once you trust the output, drop --dry-run
clearstack clean ~/Developer --categories=node_modules,.next --yes

# Diagnose what clearstack can clean on your system
clearstack doctor

# Undo the last operation
clearstack undo --last 1
```

## Safety guarantees

- First run is always a dry-run.
- Whitelisted paths (`/`, `/System`, `~`, etc.) can never be deleted.
- Symlinks are never followed.
- `pnpm store` always uses `pnpm store prune` — never raw deletion.
- `GOMODCACHE` / `GOCACHE` always use `go clean -modcache/-cache`.
- Every operation is logged to `~/.local/state/clearstack/operations.jsonl` and reversible via `clearstack undo`.
- Trash is the default strategy; `--hard` opts into real deletion.

Read [`docs/SAFETY.md`](docs/SAFETY.md) for the full safety model.

## Docs

- [`docs/INSTALL.md`](docs/INSTALL.md) — per-platform install instructions
- [`docs/CATEGORIES.md`](docs/CATEGORIES.md) — every category, paths, safety level
- [`docs/CONFIG.md`](docs/CONFIG.md) — config file reference
- [`docs/SAFETY.md`](docs/SAFETY.md) — safety model & guarantees
- [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) — contributing

## Status

🚧 **Early development.** See [`CHANGELOG.md`](CHANGELOG.md) for progress.

## License

MIT © [Guilherme Jansen](https://github.com/guilhermejansen) / Setup Automatizado
