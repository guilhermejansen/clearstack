#!/usr/bin/env bash
# clearstack installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/guilhermejansen/clearstack/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/guilhermejansen/clearstack/main/install.sh | bash -s -- --version v1.0.0
#
# Env:
#   CLEARSTACK_INSTALL_DIR — override default $HOME/.local/bin
#   CLEARSTACK_VERSION     — pin a specific version (default: latest)
#
set -euo pipefail

REPO="guilhermejansen/clearstack"
BINARY="clearstack"
INSTALL_DIR="${CLEARSTACK_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${CLEARSTACK_VERSION:-latest}"

log() { printf "\033[1;36m[clearstack]\033[0m %s\n" "$*"; }
die() { printf "\033[1;31m[clearstack] error:\033[0m %s\n" "$*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || die "missing required tool: $1"; }
need curl
need tar
need uname

os_name() {
  case "$(uname -s)" in
    Darwin) echo darwin ;;
    Linux)  echo linux ;;
    *)      die "unsupported OS: $(uname -s)" ;;
  esac
}

arch_name() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | grep -E '"tag_name":' \
      | head -n1 \
      | sed -E 's/.*"([^"]+)".*/\1/'
  else
    echo "$VERSION"
  fi
}

main() {
  local os arch version url tmp
  os="$(os_name)"
  arch="$(arch_name)"
  version="$(resolve_version)"
  [ -n "$version" ] || die "unable to resolve version"

  url="https://github.com/$REPO/releases/download/$version/${BINARY}_${version#v}_${os}_${arch}.tar.gz"
  log "installing $BINARY $version for $os/$arch"
  log "from $url"

  mkdir -p "$INSTALL_DIR"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  curl -fsSL "$url" -o "$tmp/archive.tar.gz" || die "download failed"
  tar -xzf "$tmp/archive.tar.gz" -C "$tmp"
  install -m 0755 "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"

  log "installed $BINARY -> $INSTALL_DIR/$BINARY"
  if ! echo ":$PATH:" | grep -q ":$INSTALL_DIR:"; then
    log "add to PATH: export PATH=\"$INSTALL_DIR:\$PATH\""
  fi
  "$INSTALL_DIR/$BINARY" version || true
}

main "$@"
