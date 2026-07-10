#!/usr/bin/env sh
# One-command installer for Linux/macOS:
#   curl -fsSL https://raw.githubusercontent.com/xSaVageAU/install-update-workflow-test/main/scripts/install.sh | sh
set -eu

REPO="xSaVageAU/install-update-workflow-test"
BINARY="iuw"
INSTALL_DIR="${IUW_INSTALL_DIR:-$HOME/.local/bin}"

detect_os() {
  case "$(uname -s)" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    *) echo "error: unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) echo "error: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_to() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_to() { wget -qO "$2" "$1"; }
else
  echo "error: this script needs curl or wget installed" >&2
  exit 1
fi

OS="$(detect_os)"
ARCH="$(detect_arch)"
ASSET="${BINARY}_${OS}_${ARCH}"

echo "Fetching latest release info for ${REPO}..."
RELEASE_JSON="$(fetch "https://api.github.com/repos/${REPO}/releases/latest")"

TAG=$(printf '%s' "$RELEASE_JSON" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
  echo "error: could not determine latest release tag (check that ${REPO} has a published release)" >&2
  exit 1
fi

echo "Installing ${BINARY} ${TAG} (${OS}/${ARCH})..."

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

fetch_to "https://github.com/${REPO}/releases/download/${TAG}/${ASSET}" "$TMP_DIR/$ASSET"

if fetch_to "https://github.com/${REPO}/releases/download/${TAG}/checksums.txt" "$TMP_DIR/checksums.txt" 2>/dev/null; then
  EXPECTED=$(grep " ${ASSET}\$" "$TMP_DIR/checksums.txt" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "$TMP_DIR/$ASSET" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL=$(shasum -a 256 "$TMP_DIR/$ASSET" | awk '{print $1}')
    else
      echo "warning: no sha256 tool found, skipping checksum verification" >&2
      ACTUAL="$EXPECTED"
    fi
    if [ "$ACTUAL" != "$EXPECTED" ]; then
      echo "error: checksum mismatch for $ASSET" >&2
      echo "  expected: $EXPECTED" >&2
      echo "  actual:   $ACTUAL" >&2
      exit 1
    fi
    echo "Checksum verified."
  fi
else
  echo "warning: checksums.txt not found, skipping verification" >&2
fi

mkdir -p "$INSTALL_DIR"
chmod +x "$TMP_DIR/$ASSET"
mv "$TMP_DIR/$ASSET" "$INSTALL_DIR/$BINARY"

echo "Installed to $INSTALL_DIR/$BINARY"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "NOTE: $INSTALL_DIR is not on your PATH."
    SHELL_RC=""
    case "${SHELL:-}" in
      */zsh) SHELL_RC="$HOME/.zshrc" ;;
      */bash) SHELL_RC="$HOME/.bashrc" ;;
    esac
    if [ -n "$SHELL_RC" ]; then
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$SHELL_RC"
      echo "Added $INSTALL_DIR to PATH in $SHELL_RC. Restart your shell or run: source $SHELL_RC"
    else
      echo "Add this to your shell profile:"
      echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    ;;
esac

echo ""
echo "Run '${BINARY} --version' to verify."
