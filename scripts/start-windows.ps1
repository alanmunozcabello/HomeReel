[CmdletBinding()]
param(
    [string]$MediaDir = "D:\media",
    [int]$Port = 8080
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$binary = Join-Path $root "HomeReel.exe"
if (-not (Test-Path -LiteralPath $binary)) { throw "No se encontro $binary. Ejecuta primero .\scripts\install-windows.ps1" }
if (-not (Test-Path -LiteralPath $MediaDir -PathType Container)) { throw "No existe la carpeta de medios: $MediaDir" }
$env:MEDIA_DIR = (Resolve-Path -LiteralPath $MediaDir).Path
$env:PORT = $Port
$env:DB_PATH = Join-Path $root "data\streamer.db"
Push-Location $root
try { & $binary } finally { Pop-Location }
