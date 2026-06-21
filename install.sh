#!/bin/sh
set -eu

REPO="judahpaul16/plume"
BIN="plume"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *) echo "plume: unsupported arch '$arch'; try: go install github.com/${REPO}@latest" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "plume: unsupported os '$os'; try: go install github.com/${REPO}@latest" >&2; exit 1 ;;
esac

asset="${BIN}-${os}-${arch}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"

dest="/usr/local/bin"
[ -w "$dest" ] 2>/dev/null || dest="${HOME}/.local/bin"
mkdir -p "$dest"

tmp="$(mktemp)"
echo "plume: downloading ${asset}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp" "$url"
else
  echo "plume: need curl or wget" >&2; exit 1
fi

chmod +x "$tmp"
mv "$tmp" "${dest}/${BIN}"
echo "plume: installed to ${dest}/${BIN}"

case ":${PATH}:" in
  *":${dest}:"*) ;;
  *) echo "plume: add ${dest} to PATH -> export PATH=\"\$PATH:${dest}\"" ;;
esac
