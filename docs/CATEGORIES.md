# Category reference

Every cleanup category clearstack ships is listed here with its safety level,
cleanup strategy, required sibling files (if any), and whether it is
dormancy-filtered.

Generate an up-to-date list on your machine with:

```sh
clearstack categories list
clearstack categories list --json
```

Or inspect a specific category inline via `clearstack doctor`.

## JavaScript / Node

| ID | Strategy | Safety | Dormant? | Notes |
|----|----------|--------|----------|-------|
| `node_modules` | trash | safe | ✓ | Requires `package.json` sibling |
| `next_cache` | trash | safe | — | `.next` — requires `package.json` |
| `turbo_cache` | trash | safe | — | `.turbo` |
| `nuxt_cache` | trash | safe | — | `.nuxt` — requires `package.json` |
| `svelte_kit_cache` | trash | safe | — | `.svelte-kit` — requires `package.json` |
| `parcel_cache` | trash | safe | — | `.parcel-cache` |
| `astro_cache` | trash | safe | — | `.astro` — requires `package.json` |
| `pnpm_store` | native (`pnpm store prune`) | caution | — | never rm directly |
| `bun_install_cache` | trash | safe | — | `~/.bun/install/cache` |

## Python

| ID | Strategy | Safety | Dormant? |
|----|----------|--------|----------|
| `pycache` | hard | safe | — |
| `pytest_cache` | hard | safe | — |
| `mypy_cache` | hard | safe | — |
| `ruff_cache` | hard | safe | — |
| `tox_cache` | trash | safe | — |
| `venv_dormant` | trash | caution | ✓ |
| `pip_cache` | native (`pip cache purge`) | safe | — |
| `poetry_cache` | native (`poetry cache clear --all .`) | safe | — |
| `uv_cache` | native (`uv cache clean`) | safe | — |

## Go

| ID | Strategy | Safety |
|----|----------|--------|
| `go_build_cache` | native (`go clean -cache`) | safe |
| `go_mod_cache` | native (`go clean -modcache`) | safe |
| `go_test_cache` | native (`go clean -testcache`) | safe |

## Rust

| ID | Strategy | Safety | Dormant? | Notes |
|----|----------|--------|----------|-------|
| `rust_target` | trash | safe | ✓ | Requires `Cargo.toml` |

## Java / Kotlin / Android

| ID | Strategy | Safety | Dormant? | Notes |
|----|----------|--------|----------|-------|
| `gradle_project_cache` | trash | safe | ✓ | Requires `build.gradle` |
| `gradle_project_cache_kts` | trash | safe | ✓ | Requires `build.gradle.kts` |
| `maven_target` | trash | safe | ✓ | Requires `pom.xml` |
| `android_cxx` | trash | safe | — | `.cxx` native cache |

## .NET

| ID | Strategy | Safety | Dormant? | Notes |
|----|----------|--------|----------|-------|
| `dotnet_bin` | trash | safe | ✓ | Requires sibling `*.csproj`/`*.fsproj`/`*.vbproj` |
| `dotnet_obj` | trash | safe | ✓ | idem |

## Xcode (darwin only)

| ID | Strategy | Safety |
|----|----------|--------|
| `xcode_derived_data` | trash | safe |

## Docker

All Docker detectors are scan-inert and must be invoked explicitly via
`clearstack clean --categories=<id>`.

| ID | Strategy | Safety |
|----|----------|--------|
| `docker_images` | `docker image prune -f --filter dangling=true` | safe |
| `docker_containers` | `docker container prune -f` | safe |
| `docker_build_cache` | `docker builder prune -f` | safe |
| `docker_networks` | `docker network prune -f` | safe |
| `docker_volumes` | `docker volume prune -f` | **danger** (holds data) |
