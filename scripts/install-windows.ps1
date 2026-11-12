[CmdletBinding()]
param(
    [string]$MediaDir = "D:\media",
    [int]$Port = 8080
)

$ErrorActionPreference = "Stop"

function Test-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Administrator)) {
    $elevationArgs = "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`" -MediaDir `"$MediaDir`" -Port $Port"
    # -Wait prevents the caller from trying to start the server before the
    # elevated build has finished.
    Start-Process PowerShell -Verb RunAs -ArgumentList $elevationArgs -Wait
    if (-not (Test-Path -LiteralPath (Join-Path (Split-Path -Parent $PSScriptRoot) "HomeReel.exe"))) {
        throw "La instalacion elevada no termino correctamente. Revisa la ventana que solicito permisos."
    }
    return
}

$root = Split-Path -Parent $PSScriptRoot
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { throw "Go no esta instalado y winget no esta disponible." }
    Write-Host "Instalando Go..."
    winget install --id GoLang.Go --exact --accept-package-agreements --accept-source-agreements
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw "Go fue instalado. Abre una nueva consola y vuelve a ejecutar el instalador." }
}
if (-not (Get-Command ffprobe -ErrorAction SilentlyContinue)) {
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw "ffprobe no esta instalado y winget no esta disponible. Instala FFmpeg y agregalo al PATH."
    }
    Write-Host "Instalando FFmpeg (incluye ffprobe)..."
    winget install --id Gyan.FFmpeg.Shared --exact --accept-package-agreements --accept-source-agreements
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not (Get-Command ffprobe -ErrorAction SilentlyContinue)) { throw "FFmpeg fue instalado, pero abre una nueva consola y ejecuta el instalador nuevamente." }
}

New-Item -ItemType Directory -Force -Path $MediaDir | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $root "data") | Out-Null
$settings = @{ media_dir = (Resolve-Path -LiteralPath $MediaDir).Path } | ConvertTo-Json
Set-Content -LiteralPath (Join-Path $root "data\settings.json") -Value $settings -Encoding utf8
Push-Location $root
try { go build -ldflags "-s -w" -o (Join-Path $root "HomeReel.exe") .\cmd\server } finally { Pop-Location }

if (-not (Get-NetFirewallRule -DisplayName "HomeReel LAN" -ErrorAction SilentlyContinue)) {
    New-NetFirewallRule -DisplayName "HomeReel LAN" -Direction Inbound -Protocol TCP -LocalPort $Port -Action Allow | Out-Null
}
Write-Host "Instalacion completada. Haz doble clic en HomeReel.exe para iniciar el servidor." -ForegroundColor Green
