# Configuration

clearstack reads its configuration from:

| OS | Default path |
|----|--------------|
| macOS | `~/Library/Application Support/clearstack/config.yaml` |
| Linux | `$XDG_CONFIG_HOME/clearstack/config.yaml` or `~/.config/clearstack/config.yaml` |
| Windows | `%APPDATA%\clearstack\config.yaml` |

Generate a starter file with:

```sh
clearstack config init
clearstack config path      # show effective path
clearstack config show      # print loaded YAML
```

Override the path per invocation via `--config /path/to/config.yaml`.

## Schema

```yaml
version: 1
profile: balanced                      # conservative | balanced | aggressive | fullstack

dormancy:
  min_age: 14d                         # 14d / 30d / 6h / 45m — project idle threshold
  check_git: true                      # refine mtime with `git log -1 --format=%ct`

safety:
  default_strategy: trash              # trash | hard
  require_dual_confirm_for:
    - pnpm_store_raw
    - m2_repository
    - avd
    - docker_volumes
  whitelist_paths:                     # always refused by the cleaner, no matter what
    - ~/code/production-critical/**

categories:
  enabled: []                          # if empty → every registered detector
  disabled: []                         # explicit opt-outs

roots:                                 # default roots for `clearstack` / TUI
  - ~/Developer
  - ~/code

docker:
  enabled: true
  volumes: false                       # volumes are NEVER on by default
  build_cache: true

ui:
  theme: auto                          # dark | light | auto
  default_sort: size                   # size | age | path | category
  default_filter: ""

telemetry:
  enabled: false
```

## Profiles

| Profile | Dormancy | Docker volumes | Build cache | Notes |
|---------|----------|----------------|-------------|-------|
| `conservative` | 30d | off | off | Only caches that regenerate instantly |
| `balanced` (default) | 14d | off | on | Sensible middle ground |
| `aggressive` | 7d | off (still opt-in) | on | Short dormancy window |
| `fullstack` | 14d | off | on | Identical to balanced with Docker on |

Apply a profile for a single invocation via `--profile aggressive`, or set
it permanently in `config.yaml`.

## Environment variables

clearstack honors the standard XDG directories on Linux:

- `XDG_CONFIG_HOME`
- `XDG_CACHE_HOME`
- `XDG_STATE_HOME`
- `XDG_DATA_HOME`

And on Windows:

- `APPDATA` — config
- `LOCALAPPDATA` — state + cache
