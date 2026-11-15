#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Un paquete .pkg solo puede crearse en macOS." >&2
  exit 1
fi

version="${1:-0.1.0}"
arch="${2:-$(uname -m)}"
case "$arch" in
  arm64|aarch64) goarch="arm64" ;;
  x86_64|amd64) goarch="amd64" ;;
  *) echo "Arquitectura no soportada: $arch" >&2; exit 1 ;;
esac

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
payload="$work/root"
mkdir -p "$payload/usr/local/lib/homereel" "$payload/usr/local/bin"

cd "$root"
CGO_ENABLED=0 GOOS=darwin GOARCH="$goarch" go build -ldflags="-s -w" -o "$payload/usr/local/lib/homereel/HomeReel" ./cmd/server
cp -R web migrations "$payload/usr/local/lib/homereel/"
cat > "$payload/usr/local/bin/homereel" <<'EOF'
#!/bin/sh
set -eu
data_dir="${HOME}/Library/Application Support/HomeReel"
mkdir -p "$data_dir" "$HOME/HomeReel/media"
cd /usr/local/lib/homereel
exec env MEDIA_DIR="$HOME/HomeReel/media" DB_PATH="$data_dir/streamer.db" /usr/local/lib/homereel/HomeReel "$@"
EOF
chmod 755 "$payload/usr/local/bin/homereel" "$payload/usr/local/lib/homereel/HomeReel"

mkdir -p "$root/dist"
pkgbuild --root "$payload" --scripts "$root/scripts/macos" --identifier "local.homereel.server" --version "$version" --install-location / "$root/dist/HomeReel-macos-${goarch}.pkg"
echo "Paquete creado: dist/HomeReel-macos-${goarch}.pkg"
