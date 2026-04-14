# Safety model

clearstack follows a "don't break the developer's day" philosophy. This doc
describes the layered guarantees the binary enforces and how to reason about
them.

## Guarantees

1. **Dry-run by default on first run.** Until the user's state directory
   contains the first-run marker, every `clean` acts as a dry-run regardless
   of `--yes`.
2. **Trash-over-delete.** The default strategy for every destructive category
   is to move files to the OS recycle bin, not to `rm -rf`. Cross-platform
   implementations:
   - macOS — Finder via `osascript`, shows up in the Trash.
   - Linux — freedesktop.org Trash spec in `$XDG_DATA_HOME/Trash`.
   - Windows — `Microsoft.VisualBasic.FileIO.FileSystem.DeleteDirectory`
     with `SendToRecycleBin`.
   - Fallback — archive to `<state-dir>/trash/<timestamp>/…` when no native
     trash is reachable.
3. **Whitelist of protected paths.** `/`, `/System`, `/Library`, `/usr`,
   `/bin`, `C:\`, `C:\Windows`, `C:\Program Files`, `~` (user home itself),
   and any user-supplied paths from `safety.whitelist_paths` are refused at
   the Cleaner layer regardless of strategy.
4. **Never follow symlinks.** Scanner and Sizer pass `Follow=false` to
   fastwalk. A symlink escaping the root cannot trick the scanner into
   touching paths the user did not approve.
5. **Official commands when available.** The raw filesystem strategy is
   avoided for categories that have a canonical clean:
   - pnpm store — `pnpm store prune` (raw rm would break every project).
   - Go caches — `go clean -cache / -modcache / -testcache` (files are
     read-only on disk).
   - pip / poetry / uv — their own `cache purge/clear/clean`.
   - Docker — `docker image/container/builder/network/volume prune`.
6. **Dormancy filtering.** Detectors that target valuable artifacts
   (`node_modules`, `target`, `.gradle`, `maven_target`, `.venv`, dotnet
   bin/obj) only report matches in projects whose last mtime is older than
   the configured threshold (14 days by default). Optional git `log -1`
   refinement uses the last commit timestamp.
7. **Required sibling files.** Directory-name detectors only match when a
   project-scope signature is present next to them (e.g., `node_modules`
   requires `package.json`, `target` requires `Cargo.toml`, Maven `target`
   requires `pom.xml`, `.NET` bin/obj requires a `*.csproj`/`*.fsproj`/
   `*.vbproj` sibling).
8. **Dual confirmation for dangerous categories.** Categories marked
   `danger` (Docker volumes, pnpm store raw, `~/.m2`, `.android/avd`) refuse
   to run without an explicit `--categories=<id> --yes` combination.
9. **Append-only journal.** Every attempted clean — dry-run, success, or
   failure — is appended to `<state-dir>/operations.jsonl` with the category,
   strategy, bytes freed, and undo reference when trash-based.
10. **Undo.** `clearstack undo` reads the journal and, for trash-based
    entries, can surface the original locations for manual restore. Full
    automated restore lands alongside the TUI confirmation screens.

## What clearstack will never do

- Touch a path that is not absolute and inside a root you passed in.
- Follow a symlink out of the declared root.
- Run as root / with sudo (even if available).
- Delete the user's home directory itself.
- Delete `pnpm-store/` via `rm` — the only strategy is `pnpm store prune`.
- Delete Docker volumes without `--categories=docker_volumes --yes`.

## Reporting a safety bug

If you find a case where clearstack was about to touch something it should
not have, please open an issue titled `SAFETY:` with the exact command and
the scenario — we treat safety bugs as P0 and will ship a patch release.
