#!/usr/bin/env bash
# install.sh — one-shot installer for scripture-mcp.
#
# Downloads the right release binary for your OS/arch, places it on PATH,
# detects which coding agents are installed locally (claude, codex), and
# wires each by calling `scripture-mcp init --target=<agent>`.
#
# Quick start:
#   curl -fsSL https://raw.githubusercontent.com/MiaoDX/verse-driven/main/install.sh | bash
#
# See `install.sh --help` for all flags.

set -euo pipefail

REPO="MiaoDX/verse-driven"
BINARY="scripture-mcp"
DEFAULT_PREFIX="${HOME}/.local/bin"

# Env overrides (mostly for tests; also useful behind a corporate mirror):
#   RELEASE_BASE_URL  — base for release tarballs (default: github.com releases).
#   LATEST_RELEASE_URL — URL whose final-redirect path ends in the latest tag.
RELEASE_BASE_URL="${RELEASE_BASE_URL:-https://github.com/${REPO}/releases/download}"
LATEST_RELEASE_URL="${LATEST_RELEASE_URL:-https://github.com/${REPO}/releases/latest}"

VERSION=""
PREFIX=""
ARCHIVE=""
NO_WIRE=0
UNINSTALL=0
ASSUME_YES=0

usage() {
  cat <<'EOF'
install.sh — install scripture-mcp and wire it into your coding agents.

Usage: install.sh [flags]

Flags:
  --version <tag>      Pin to a specific release tag (default: latest).
  --prefix <dir>       Install directory (default: $HOME/.local/bin).
  --from-archive <p>   Skip the network and unpack a local .tar.gz instead.
  --no-wire            Install the binary only; skip agent wiring.
  --uninstall          Remove wiring (init --uninstall) and the binary.
  --yes, -y            Wire all detected agents without prompting.
  -h, --help           Show this help and exit.

Acceptance criteria from issue #7:
  - Detects OS/arch, downloads the matching release, places it on PATH.
  - Detects claude / codex on PATH and offers to wire each.
  - Calls `scripture-mcp init --target=...` per detected agent.
  - Re-running upgrades the binary; init is idempotent on configs.
  - Uninstall path: `scripture-mcp init --uninstall --target=...`.
EOF
}

log() { printf 'install.sh: %s\n' "$*"; }
warn() { printf 'install.sh: %s\n' "$*" >&2; }
die() { warn "$*"; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:-}"; shift 2 ;;
    --version=*) VERSION="${1#*=}"; shift ;;
    --prefix) PREFIX="${2:-}"; shift 2 ;;
    --prefix=*) PREFIX="${1#*=}"; shift ;;
    --from-archive) ARCHIVE="${2:-}"; shift 2 ;;
    --from-archive=*) ARCHIVE="${1#*=}"; shift ;;
    --no-wire) NO_WIRE=1; shift ;;
    --uninstall) UNINSTALL=1; shift ;;
    --yes|-y) ASSUME_YES=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) warn "unknown flag: $1"; usage >&2; exit 2 ;;
  esac
done

PREFIX="${PREFIX:-$DEFAULT_PREFIX}"

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux)  echo "linux" ;;
    *) die "unsupported OS: $(uname -s) (install.sh supports macOS and Linux only)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    arm64|aarch64) echo "arm64" ;;
    x86_64|amd64)  echo "x86_64" ;;
    *) die "unsupported arch: $(uname -m)" ;;
  esac
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    printf '%s\n' "$VERSION"
    return
  fi
  if ! command -v curl >/dev/null 2>&1; then
    die "curl is required to resolve the latest release (or pass --version <tag>)"
  fi
  # GitHub redirects /releases/latest to /releases/tag/<tag>; capture the
  # effective URL and strip everything up to the last slash.
  local effective tag
  effective="$(curl -fsSL -o /dev/null -w '%{url_effective}' "$LATEST_RELEASE_URL")"
  tag="${effective##*/}"
  if [ -z "$tag" ] || [ "$tag" = "latest" ]; then
    die "could not resolve latest release from $LATEST_RELEASE_URL (pass --version <tag>)"
  fi
  printf '%s\n' "$tag"
}

fetch() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$dest"
  else
    die "need curl or wget to download $url"
  fi
}

# install_binary downloads the release tarball (or uses --from-archive),
# extracts the binary, and copies it to ${PREFIX}/${BINARY}. Re-running
# overwrites the previous binary (the upgrade path).
install_binary() {
  local os arch tmp src
  os="$(detect_os)"
  arch="$(detect_arch)"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN

  if [ -n "$ARCHIVE" ]; then
    [ -f "$ARCHIVE" ] || die "--from-archive: file not found: $ARCHIVE"
    log "unpacking local archive $ARCHIVE"
    src="$ARCHIVE"
  else
    local version asset url
    version="$(resolve_version)"
    asset="${BINARY}-${version}-${os}-${arch}.tar.gz"
    url="${RELEASE_BASE_URL}/${version}/${asset}"
    log "downloading $asset"
    src="${tmp}/${asset}"
    fetch "$url" "$src"
  fi

  tar -xzf "$src" -C "$tmp"
  [ -f "${tmp}/${BINARY}" ] || die "archive does not contain ${BINARY} at the top level"

  mkdir -p "$PREFIX"
  cp "${tmp}/${BINARY}" "${PREFIX}/${BINARY}.new"
  chmod 0755 "${PREFIX}/${BINARY}.new"
  mv -f "${PREFIX}/${BINARY}.new" "${PREFIX}/${BINARY}"
  log "installed ${PREFIX}/${BINARY}"

  case ":$PATH:" in
    *":${PREFIX}:"*) ;;
    *) warn "note: ${PREFIX} is not on \$PATH. Add this to your shell rc:"
       warn "    export PATH=\"${PREFIX}:\$PATH\"" ;;
  esac
}

remove_binary() {
  local target="${PREFIX}/${BINARY}"
  if [ -f "$target" ]; then
    rm -f "$target"
    log "removed $target"
  else
    log "nothing to remove at $target"
  fi
}

have_agent() { command -v "$1" >/dev/null 2>&1; }

# prompt_yes returns 0 (yes) when --yes was passed, when no TTY is
# available (typical curl|bash session — default to wiring), or when the
# user answers y/Y/yes at the prompt. Reading from /dev/tty lets us
# prompt even when stdin is the curl pipe.
prompt_yes() {
  if [ "$ASSUME_YES" = "1" ]; then return 0; fi
  if [ ! -t 0 ] && [ ! -e /dev/tty ]; then return 0; fi
  local reply=""
  printf 'install.sh: %s [Y/n] ' "$1" >&2
  if [ -t 0 ]; then
    read -r reply || reply=""
  else
    read -r reply </dev/tty || reply=""
  fi
  case "${reply:-y}" in
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

wire_agents() {
  local bin="${PREFIX}/${BINARY}"
  local any=0
  if have_agent claude; then
    any=1
    if prompt_yes "wire Claude Code (~/.claude/settings.json)?"; then
      "$bin" init --target=claude-code
    else
      log "skipped Claude Code wiring"
    fi
  fi
  if have_agent codex; then
    any=1
    if prompt_yes "wire Codex (~/.codex/config.toml)?"; then
      "$bin" init --target=codex
    else
      log "skipped Codex wiring"
    fi
  fi
  if [ "$any" = "0" ]; then
    log "no supported agents detected on \$PATH (claude, codex)"
    log "install one and re-run, or invoke 'scripture-mcp init --target=...' manually"
  fi
}

unwire_agents() {
  local bin="${PREFIX}/${BINARY}"
  if [ ! -x "$bin" ]; then
    log "${bin} not found; skipping config cleanup"
    return
  fi
  if have_agent claude; then
    "$bin" init --uninstall --target=claude-code || true
  fi
  if have_agent codex; then
    "$bin" init --uninstall --target=codex || true
  fi
}

if [ "$UNINSTALL" = "1" ]; then
  unwire_agents
  remove_binary
  log "done"
  exit 0
fi

install_binary
if [ "$NO_WIRE" = "0" ]; then
  wire_agents
fi
log "done"
