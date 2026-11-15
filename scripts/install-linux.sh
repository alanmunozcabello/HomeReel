#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if command -v apt-get >/dev/null 2>&1; then
  sudo apt-get update
  sudo apt-get install -y ffmpeg golang-go
else
  echo "Instala Go y FFmpeg con el gestor de paquetes de tu distribucion." >&2
  exit 1
fi

cd "$root"
mkdir -p data media
CGO_ENABLED=0 go build -ldflags="-s -w" -o homereel ./cmd/server
chmod +x homereel
echo "Listo. Ejecuta: ./homereel (usara $root/media por defecto)"
