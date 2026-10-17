#Requires -Version 5.1
<#
.SYNOPSIS
  Start Momentos backends + Expo for testing on a physical phone.

.DESCRIPTION
  1. Detects your LAN IPv4 so the phone can reach the gateway (localhost will not work on device).
  2. Ensures apps/mobile/.env has EXPO_PUBLIC_GATEWAY_URL=http://<lan-ip>:8080
  3. Starts gateway, moment, media, render, and ml worker in separate PowerShell windows.
  4. Starts Expo with --lan (same Wi-Fi) or --tunnel (off-network / flaky LAN).

.PARAMETER Tunnel
  Use Expo tunnel instead of LAN (slower, works across networks).

.PARAMETER SkipBackend
  Only start Expo (backends already running).

.EXAMPLE
  .\scripts\dev-phone.ps1
  .\scripts\dev-phone.ps1 -Tunnel
#>
param(
  [switch]$Tunnel,
  [switch]$SkipBackend
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root

function Get-LanIPv4 {
  $candidates = Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
    Where-Object {
      $_.IPAddress -notlike "127.*" -and
      $_.IPAddress -notlike "169.254.*" -and
      $_.PrefixOrigin -ne "WellKnown"
    } |
    Sort-Object -Property InterfaceMetric

  if ($candidates) {
    return $candidates[0].IPAddress
  }

  # Fallback: parse ipconfig
  $m = ipconfig | Select-String -Pattern "IPv4.*:\s*([\d.]+)" | ForEach-Object {
    if ($_.Matches.Groups[1].Value -notmatch "^(127\.|169\.254\.)") {
      $_.Matches.Groups[1].Value
    }
  } | Select-Object -First 1
  if (-not $m) { throw "Could not detect LAN IPv4. Connect to Wi-Fi and retry." }
  return $m
}

function Ensure-MobileEnv([string]$GatewayUrl) {
  $envPath = Join-Path $Root "apps\mobile\.env"
  $example = Join-Path $Root "apps\mobile\.env.example"
  if (-not (Test-Path $envPath)) {
    if (Test-Path $example) {
      Copy-Item $example $envPath
      Write-Host "Created apps/mobile/.env from .env.example"
    } else {
      @"
EXPO_PUBLIC_GATEWAY_URL=$GatewayUrl
EXPO_PUBLIC_SUPABASE_URL=
EXPO_PUBLIC_SUPABASE_ANON_KEY=
"@ | Set-Content -Path $envPath -Encoding utf8
    }
  }

  $lines = Get-Content $envPath
  $found = $false
  $out = foreach ($line in $lines) {
    if ($line -match "^\s*EXPO_PUBLIC_GATEWAY_URL\s*=") {
      $found = $true
      "EXPO_PUBLIC_GATEWAY_URL=$GatewayUrl"
    } else {
      $line
    }
  }
  if (-not $found) {
    $out = @($out) + "EXPO_PUBLIC_GATEWAY_URL=$GatewayUrl"
  }
  $out | Set-Content -Path $envPath -Encoding utf8
  Write-Host "EXPO_PUBLIC_GATEWAY_URL=$GatewayUrl"
}

function Start-ServiceWindow([string]$Title, [string]$WorkingDir, [string]$Command) {
  $escaped = $Command -replace '"', '\"'
  Start-Process powershell -ArgumentList @(
    "-NoExit",
    "-Command",
    "Set-Location '$WorkingDir'; Write-Host '=== $Title ===' -ForegroundColor Cyan; $escaped"
  ) | Out-Null
  Write-Host "Started: $Title"
}

# Refresh PATH so Go is visible in child shells
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
  [System.Environment]::GetEnvironmentVariable("Path", "User")

$lan = Get-LanIPv4
$gatewayUrl = "http://${lan}:8080"
Ensure-MobileEnv $gatewayUrl

Write-Host ""
Write-Host "Phone setup" -ForegroundColor Green
Write-Host "  1. Phone and PC on the same Wi-Fi (unless -Tunnel)."
Write-Host "  2. Install Expo Go on your phone."
Write-Host "  3. Fill Supabase keys in apps/mobile/.env if you have not already."
Write-Host "  4. Scan the QR code from the Expo window."
Write-Host "  Gateway from phone: $gatewayUrl"
Write-Host ""

if (-not $SkipBackend) {
  $goCmd = Get-Command go -ErrorAction SilentlyContinue
  if (-not $goCmd) {
    $goCandidate = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $goCandidate) {
      $env:Path = "C:\Program Files\Go\bin;" + $env:Path
    } else {
      throw "Go not found on PATH. Install Go or open a terminal where 'go version' works."
    }
  }

  Start-ServiceWindow "momentos-gateway :8080" (Join-Path $Root "services\gateway") `
    "go run ./cmd/gateway"

  Start-ServiceWindow "momentos-moment-service :8081" (Join-Path $Root "services\moment-service") `
    "`$env:MEDIA_SKIP_VALIDATE='true'; go run ./cmd/moment-service"

  Start-ServiceWindow "momentos-media-service :8082" (Join-Path $Root "services\media-service") `
    "go run ./cmd/media-service"

  Start-ServiceWindow "momentos-render-service :8083" (Join-Path $Root "services\render-service") `
    "`$env:MOMENT_SERVICE_URL='http://127.0.0.1:8081'; go run ./cmd/render-service"

  $mlDir = Join-Path $Root "ml"
  $mlCmd = if (Get-Command uv -ErrorAction SilentlyContinue) {
    "`$env:CUT_PLANNER='stub'; `$env:MOMENT_SERVICE_URL='http://127.0.0.1:8081'; uv sync; uv run python -m momentos_ml.worker"
  } else {
    "`$env:CUT_PLANNER='stub'; `$env:MOMENT_SERVICE_URL='http://127.0.0.1:8081'; python -m momentos_ml.worker"
  }
  Start-ServiceWindow "momentos-ml-worker" $mlDir $mlCmd

  Write-Host "Waiting a few seconds for backends to bind..."
  Start-Sleep -Seconds 3
}

$mobileDir = Join-Path $Root "apps\mobile"
$expoFlag = if ($Tunnel) { "--tunnel" } else { "--lan" }

# Prefer local expo binary via npx/npm
$expoCmd = "npx expo start $expoFlag --clear"
if (Test-Path (Join-Path $mobileDir "node_modules\expo\package.json")) {
  $expoCmd = "npx expo start $expoFlag --clear"
} else {
  Write-Host "Installing mobile deps (npm install)..."
  Push-Location $mobileDir
  npm install
  Pop-Location
}

Start-ServiceWindow "momentos-expo $expoFlag" $mobileDir $expoCmd

Write-Host ""
Write-Host "All set. Open the Expo window, scan the QR with Expo Go." -ForegroundColor Green
Write-Host "If the app cannot reach the API, confirm Windows Firewall allows ports 8080-8083,"
Write-Host "or re-run with:  .\scripts\dev-phone.ps1 -Tunnel"
