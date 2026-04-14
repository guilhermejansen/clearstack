# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Sprint 7: docs (`docs/SAFETY.md`, `docs/CATEGORIES.md`, `docs/CONFIG.md`,
  `docs/INSTALL.md`, `docs/DEVELOPMENT.md`).
- Sprint 6: install.sh and GitHub Actions release workflow.
- Sprint 5: Windows Recycle Bin trasher (PowerShell backend), Java/Kotlin
  (.gradle, maven target, .cxx), .NET (bin/obj) detectors.
- Sprint 4: Bubble Tea TUI (scanning, results, confirming, cleaning,
  summary states) with Catppuccin Mocha theme, multi-select, fuzzy filter,
  sort rotation, help overlay, and npkill+vim keybindings.
- Sprint 3: Docker CLI-backed detectors — images, containers, build cache,
  networks, volumes (volumes gated as danger).
- Sprint 2: Cobra/Viper CLI (scan/clean/analyze/doctor/config/undo/
  categories/completion/version), config schema with profiles, native
  command detectors for pnpm/bun/pip/poetry/uv, safety net with dry-run
  default and interactive confirm, JSON output everywhere.
- Sprint 1: Core engine (fastwalk scanner, concurrent sizer, classifier,
  dormancy policy with optional git log timestamps, cleaner with trash/
  hard/native-command strategies, operations journal, multi-OS trash),
  base detectors for Node, Python, Go, Rust, and Xcode.
- Sprint 0: repository bootstrap (structure, go.mod, license, CI skeleton,
  Makefile, goreleaser config, placeholder `clearstack version` command).
