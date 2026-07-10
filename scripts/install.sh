#!/usr/bin/env sh
# One-command installer for Linux/macOS:
#   curl -fsSL https://raw.githubusercontent.com/xSaVageAU/install-update-workflow-test/main/scripts/install.sh | sh
set -eu

REPO="xSaVageAU/install-update-workflow-test"
BINARY="iuw"
INSTALL_DIR="${IUW_INSTALL_DIR:-$HOME/.local/bin}"

# Color output when connected to a terminal; respects NO_COLOR (https://no-color.org).
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  c_reset='\033[0m'; c_blue='\033[1;34m'; c_green='\033[1;32m'; c_yellow='\033[1;33m'; c_red='\033[1;31m'
else
  c_reset=''; c_blue=''; c_green=''; c_yellow=''; c_red=''
fi

info()    { printf '%b\n' "${c_blue}==>${c_reset} $*"; }
success() { printf '%b\n' "${c_green}==>${c_reset} $*"; }
note()    { printf '%b\n' "${c_yellow}$*${c_reset}"; }
warn()    { printf '%b\n' "${c_yellow}warning:${c_reset} $*" >&2; }
err()     { printf '%b\n' "${c_red}error:${c_reset} $*" >&2; }

detect_os() {
  case "$(uname -s)" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    *) err "unsupported OS: $(uname -s)"; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) err "unsupported architecture: $(uname -m)"; exit 1 ;;
  esac
}

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_to() { curl -fsSL -o "$2" "$1"; }
  fetch_to_progress() { curl -fL --progress-bar -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_to() { wget -qO "$2" "$1"; }
  fetch_to_progress() { wget -q --show-progress -O "$2" "$1"; }
else
  err "this script needs curl or wget installed"
  exit 1
fi

OS="$(detect_os)"
ARCH="$(detect_arch)"
ASSET="${BINARY}_${OS}_${ARCH}"

info "Fetching latest release info for ${REPO}..."
RELEASE_JSON="$(fetch "https://api.github.com/repos/${REPO}/releases/latest")"

TAG=$(printf '%s' "$RELEASE_JSON" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
  err "could not determine latest release tag (check that ${REPO} has a published release)"
  exit 1
fi

info "Installing ${BINARY} ${TAG} (${OS}/${ARCH})..."

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

fetch_to_progress "https://github.com/${REPO}/releases/download/${TAG}/${ASSET}" "$TMP_DIR/$ASSET"

if fetch_to "https://github.com/${REPO}/releases/download/${TAG}/checksums.txt" "$TMP_DIR/checksums.txt" 2>/dev/null; then
  EXPECTED=$(grep " ${ASSET}\$" "$TMP_DIR/checksums.txt" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "$TMP_DIR/$ASSET" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL=$(shasum -a 256 "$TMP_DIR/$ASSET" | awk '{print $1}')
    else
      warn "no sha256 tool found, skipping checksum verification"
      ACTUAL=""
    fi
    if [ -n "$ACTUAL" ]; then
      if [ "$ACTUAL" != "$EXPECTED" ]; then
        err "checksum mismatch for $ASSET"
        err "  expected: $EXPECTED"
        err "  actual:   $ACTUAL"
        exit 1
      fi
      success "Checksum verified."
    fi
  fi
else
  warn "checksums.txt not found, skipping verification"
fi

mkdir -p "$INSTALL_DIR"
chmod +x "$TMP_DIR/$ASSET"
mv "$TMP_DIR/$ASSET" "$INSTALL_DIR/$BINARY"

success "Installed to $INSTALL_DIR/$BINARY"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    note "NOTE: $INSTALL_DIR is not on your PATH."
    SHELL_RC=""
    case "${SHELL:-}" in
      */zsh) SHELL_RC="$HOME/.zshrc" ;;
      */bash) SHELL_RC="$HOME/.bashrc" ;;
    esac
    if [ -n "$SHELL_RC" ]; then
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$SHELL_RC"
      success "Added $INSTALL_DIR to PATH in $SHELL_RC. Restart your shell or run: source $SHELL_RC"
    else
      note "Add this to your shell profile:"
      echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    ;;
esac

echo ""
info "Run '${BINARY} --version' to verify."
