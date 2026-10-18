#Requires -Version 5.1
<#
.SYNOPSIS
  Start Momentos backends + Expo for testing on a physical phone.

.DESCRIPTION
  Fresh restart: stops any prior phone-stack processes, then starts clean.
  1. Detects your LAN IPv4 so the phone can reach the gateway (localhost will not work on device).
  2. Ensures apps/mobile/.env has EXPO_PUBLIC_GATEWAY_URL=http://<lan-ip>:8080
  3. Starts gateway, moment, media, render, and ml worker in separate PowerShell windows.
  4. Starts Expo with --lan (same Wi-Fi) or --tunnel (off-network / flaky LAN).
  Postgres Docker container is left running (not wiped).

.PARAMETER Tunnel
  Use Expo tunnel instead of LAN (slower, works across networks).

.PARAMETER SkipBackend
  Only restart Expo (leave backends alone).

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

function Import-DotEnv([string]$Path) {
  if (-not (Test-Path $Path)) {
    throw "Missing $Path - copy .env.example to .env and fill values."
  }
  Get-Content $Path | ForEach-Object {
    $line = $_.Trim()
    if (-not $line -or $line.StartsWith("#")) { return }
    $eq = $line.IndexOf("=")
    if ($eq -lt 1) { return }
    $key = $line.Substring(0, $eq).Trim()
    $value = $line.Substring($eq + 1).Trim()
    if (
      ($value.StartsWith('"') -and $value.EndsWith('"')) -or
      ($value.StartsWith("'") -and $value.EndsWith("'"))
    ) {
      $value = $value.Substring(1, $value.Length - 2)
    }
    Set-Item -Path "Env:$key" -Value $value
  }
  Write-Host "Loaded env from $Path"
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
EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY=
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
    "Set-Location '$WorkingDir'; Write-Host '=== $Title ===' -ForegroundColor Cyan; `$Host.UI.RawUI.WindowTitle = '$Title'; $escaped"
  ) | Out-Null
  Write-Host "Started: $Title"
}

function Stop-ProcessTree([int]$ProcessId) {
  if ($ProcessId -le 0) { return }
  if ($ProcessId -eq $PID) { return }
  Get-CimInstance Win32_Process -Filter "ParentProcessId=$ProcessId" -ErrorAction SilentlyContinue |
    ForEach-Object { Stop-ProcessTree -ProcessId ([int]$_.ProcessId) }
  Stop-Process -Id $ProcessId -Force -ErrorAction SilentlyContinue
}

function Stop-PhoneStack {
  param([switch]$ExpoOnly)

  Write-Host "Stopping prior phone-stack processes..."

  $ports = @(19000, 19001, 8085)
  if (-not $ExpoOnly) {
    $ports = @(8080, 8081, 8082, 8083) + $ports
  }

  $pids = New-Object 'System.Collections.Generic.HashSet[int]'
  foreach ($port in $ports) {
    Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue |
      ForEach-Object { [void]$pids.Add([int]$_.OwningProcess) }
  }

  $cmdPatterns = @(
    'npx expo start',
    'expo start',
    'node_modules[\\/]expo[\\/]',
    'momentos-expo'
  )
  if (-not $ExpoOnly) {
    $cmdPatterns += @(
      'go run \./cmd/gateway',
      'go run \./cmd/moment-service',
      'go run \./cmd/media-service',
      'go run \./cmd/render-service',
      'momentos_ml\.worker',
      'momentos-gateway',
      'momentos-moment-service',
      'momentos-media-service',
      'momentos-render-service',
      'momentos-ml-worker'
    )
  }

  Get-CimInstance Win32_Process -ErrorAction SilentlyContinue | ForEach-Object {
    $cmd = $_.CommandLine
    if (-not $cmd) { return }
    foreach ($pat in $cmdPatterns) {
      if ($cmd -match $pat) {
        [void]$pids.Add([int]$_.ProcessId)
        break
      }
    }
  }

  Get-Process -Name powershell, pwsh, node, go -ErrorAction SilentlyContinue | ForEach-Object {
    $title = $_.MainWindowTitle
    if (-not $title) { return }
    if ($title -match '^momentos-(gateway|moment-service|media-service|render-service|ml-worker|expo)\b') {
      if ($ExpoOnly -and $title -notmatch '^momentos-expo\b') { return }
      [void]$pids.Add([int]$_.Id)
    }
  }

  foreach ($procId in $pids) {
    Stop-ProcessTree -ProcessId $procId
  }

  Start-Sleep -Milliseconds 500
  Write-Host "Prior phone-stack cleared."
}

function Test-PostgresReady {
  param([string]$ComposeFile)
  $prev = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    & docker compose -f $ComposeFile exec -T postgres pg_isready -U postgres -d momentos 2>$null | Out-Null
    return ($LASTEXITCODE -eq 0)
  } finally {
    $ErrorActionPreference = $prev
  }
}

function Wait-PostgresReady {
  param(
    [string]$ComposeFile,
    [int]$TimeoutSec = 90
  )
  # Require several consecutive successes so we do not race a post-init restart.
  $need = 3
  $ok = 0
  for ($i = 0; $i -lt $TimeoutSec; $i++) {
    if (Test-PostgresReady -ComposeFile $ComposeFile) {
      $ok++
      if ($ok -ge $need) { return }
    } else {
      $ok = 0
    }
    Start-Sleep -Seconds 1
  }
  throw "Postgres did not become ready on localhost:5432."
}

function Invoke-SqlFile {
  param(
    [string]$ComposeFile,
    [string]$SqlPath,
    [int]$Attempts = 5
  )
  $name = Split-Path $SqlPath -Leaf
  for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
    Write-Host "Applying $name (attempt $attempt/$Attempts)..."
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
      Get-Content -Path $SqlPath -Raw | & docker compose -f $ComposeFile exec -T postgres `
        psql -U postgres -d momentos -v ON_ERROR_STOP=1
      $sqlExit = $LASTEXITCODE
    } finally {
      $ErrorActionPreference = $prev
    }
    if ($sqlExit -eq 0) { return }
    if ($attempt -eq $Attempts) {
      throw "Failed applying $SqlPath"
    }
    Write-Host "Migration failed (Postgres may still be starting); retrying..."
    Start-Sleep -Seconds 2
    Wait-PostgresReady -ComposeFile $ComposeFile -TimeoutSec 30
  }
}

function Invoke-DockerCompose {
  # docker compose writes progress to stderr; with $ErrorActionPreference=Stop
  # PowerShell turns that into a terminating NativeCommandError.
  # Pass args as an explicit array (not splatted switches): PowerShell otherwise
  # eats bare -d / -f when using ValueFromRemainingArguments.
  param([Parameter(Mandatory = $true)][string[]]$ComposeArgs)
  $prev = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    & docker @ComposeArgs
    return $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $prev
  }
}

function Ensure-Postgres {
  $docker = Get-Command docker -ErrorAction SilentlyContinue
  if (-not $docker) {
    throw "Docker not found. Install Docker Desktop, or run Postgres yourself and set DATABASE_URL in .env."
  }

  $composeFile = Join-Path $Root "docker-compose.yml"
  Write-Host "Starting Postgres (docker compose)..."
  $upExit = Invoke-DockerCompose -ComposeArgs @(
    "compose", "-f", $composeFile, "up", "-d", "postgres"
  )
  if ($upExit -ne 0) {
    throw "docker compose up failed (exit $upExit). Is Docker Desktop running?"
  }

  Wait-PostgresReady -ComposeFile $composeFile

  # Idempotent migrations (script-owned; not via docker-entrypoint-initdb.d).
  $migrations = @(
    (Join-Path $Root "services\moment-service\migrations\001_init.sql"),
    (Join-Path $Root "services\media-service\migrations\001_init.sql")
  )
  foreach ($sqlPath in $migrations) {
    Invoke-SqlFile -ComposeFile $composeFile -SqlPath $sqlPath
  }
  Write-Host "Postgres ready at localhost:5432 (db=momentos)"
}

# Refresh PATH so Go is visible in child shells
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
  [System.Environment]::GetEnvironmentVariable("Path", "User")

Import-DotEnv (Join-Path $Root ".env")

$lan = Get-LanIPv4
$gatewayUrl = "http://${lan}:8080"
Ensure-MobileEnv $gatewayUrl

Write-Host ""
Write-Host "Phone setup (fresh restart)" -ForegroundColor Green
Write-Host "  1. Phone and PC on the same Wi-Fi (unless -Tunnel)."
Write-Host "  2. Expo Go SDK must match apps/mobile (see package.json expo version)."
Write-Host "  3. Fill Supabase keys in apps/mobile/.env if you have not already."
Write-Host "  4. Scan the QR code from the Expo window."
Write-Host "  Gateway from phone: $gatewayUrl"
Write-Host ""

if ($SkipBackend) {
  Stop-PhoneStack -ExpoOnly
} else {
  Stop-PhoneStack

  if (-not $env:DATABASE_URL) {
    throw "DATABASE_URL is missing in .env (required by moment-service / media-service)."
  }

  Ensure-Postgres

  $goCmd = Get-Command go -ErrorAction SilentlyContinue
  if (-not $goCmd) {
    $goCandidate = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $goCandidate) {
      $env:Path = "C:\Program Files\Go\bin;" + $env:Path
    } else {
      throw "Go not found on PATH. Install Go or open a terminal where 'go version' works."
    }
  }

  # Refresh PATH so a just-installed winget ffmpeg is visible.
  $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
    [System.Environment]::GetEnvironmentVariable("Path", "User")

  $ffmpegBin = $null
  if ($env:RENDER_FFMPEG_PATH -and (Test-Path -LiteralPath $env:RENDER_FFMPEG_PATH)) {
    $ffmpegBin = $env:RENDER_FFMPEG_PATH
  } else {
    $ffmpegCmd = Get-Command ffmpeg -ErrorAction SilentlyContinue
    if ($ffmpegCmd) { $ffmpegBin = $ffmpegCmd.Source }
  }
  if (-not $ffmpegBin) {
    $wingetRoot = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
    if (Test-Path $wingetRoot) {
      $hit = Get-ChildItem -Path $wingetRoot -Filter ffmpeg.exe -Recurse -ErrorAction SilentlyContinue |
        Select-Object -First 1
      if ($hit) { $ffmpegBin = $hit.FullName }
    }
  }
  if (-not $ffmpegBin) {
    throw "ffmpeg not found. Install with: winget install --id Gyan.FFmpeg -e   (or set RENDER_FFMPEG_PATH)"
  }
  $env:RENDER_FFMPEG_PATH = $ffmpegBin
  Write-Host "Using ffmpeg: $ffmpegBin"

  # Child windows inherit this process env (DATABASE_URL, tokens, Supabase, etc).
  Start-ServiceWindow "momentos-gateway :8080" (Join-Path $Root "services\gateway") `
    "go run ./cmd/gateway"

  # MEDIA_SKIP_VALIDATE=false + MEDIA_SERVICE_URL so cut-draft/render get signed storageUrl.
  $mediaURL = if ($env:MEDIA_SERVICE_URL) { $env:MEDIA_SERVICE_URL } else { "http://127.0.0.1:8082" }
  $mediaURLEsc = $mediaURL.Replace("'", "''")
  Start-ServiceWindow "momentos-moment-service :8081" (Join-Path $Root "services\moment-service") `
    "`$env:MEDIA_SKIP_VALIDATE='false'; `$env:MEDIA_SERVICE_URL='$mediaURLEsc'; go run ./cmd/moment-service"

  Start-ServiceWindow "momentos-media-service :8082" (Join-Path $Root "services\media-service") `
    "go run ./cmd/media-service"

  # RENDER_EDIT_AUDIO_PATH falls back to EDIT_AUDIO_PATH from root .env when unset.
  $renderEdit = if ($env:RENDER_EDIT_AUDIO_PATH) { $env:RENDER_EDIT_AUDIO_PATH } else { $env:EDIT_AUDIO_PATH }
  $ffmpegEsc = $ffmpegBin.Replace("'", "''")
  $renderEnv = "`$env:MOMENT_SERVICE_URL='http://127.0.0.1:8081'; `$env:RENDER_FFMPEG_PATH='$ffmpegEsc'"
  if ($renderEdit) {
    $escaped = $renderEdit.Replace("'", "''")
    $renderEnv = "$renderEnv; `$env:RENDER_EDIT_AUDIO_PATH='$escaped'"
  }
  Start-ServiceWindow "momentos-render-service :8083" (Join-Path $Root "services\render-service") `
    "$renderEnv; go run ./cmd/render-service"

  $mlDir = Join-Path $Root "ml"
  # VISION_SCORING=metadata by default — Gemini placements do not use per-photo vision scores.
  if (-not $env:VISION_SCORING) { $env:VISION_SCORING = "metadata" }
  $visionEsc = $env:VISION_SCORING.Replace("'", "''")
  $mlCmd = if (Get-Command uv -ErrorAction SilentlyContinue) {
    "`$env:MOMENT_SERVICE_URL='http://127.0.0.1:8081'; `$env:VISION_SCORING='$visionEsc'; uv sync --extra dev; uv run python -m momentos_ml.worker"
  } else {
    "`$env:MOMENT_SERVICE_URL='http://127.0.0.1:8081'; `$env:VISION_SCORING='$visionEsc'; python -m momentos_ml.worker"
  }
  Start-ServiceWindow "momentos-ml-worker" $mlDir $mlCmd

  Write-Host "Waiting a few seconds for backends to bind..."
  Start-Sleep -Seconds 3
}

$mobileDir = Join-Path $Root "apps\mobile"
$expoFlag = if ($Tunnel) { "--tunnel" } else { "--lan" }

$needsInstall = -not (Test-Path (Join-Path $mobileDir "node_modules\expo\package.json")) -or
  -not (Test-Path (Join-Path $mobileDir "node_modules\expo-asset\package.json"))
if ($needsInstall) {
  Write-Host "Installing mobile deps..."
  # Prefer npm when this app already has a package-lock (avoids pnpm/npm node_modules fights).
  if ((Test-Path (Join-Path $mobileDir "package-lock.json")) -or -not (Get-Command pnpm -ErrorAction SilentlyContinue)) {
    Push-Location $mobileDir
    npm install
    Pop-Location
  } else {
    pnpm --filter @momentos/mobile install
  }
}

# Metro must not use 8081 — that is moment-service (MOMENT_SERVICE_ADDR).
$expoCmd = "npx expo start $expoFlag --port 8085 --clear"
Start-ServiceWindow "momentos-expo $expoFlag" $mobileDir $expoCmd

Write-Host ""
Write-Host "All set. Open the Expo window, scan the QR with Expo Go." -ForegroundColor Green
Write-Host "If the app cannot reach the API, confirm Windows Firewall allows ports 8080-8083,"
Write-Host "or re-run with:  .\scripts\dev-phone.ps1 -Tunnel"
