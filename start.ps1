# Starts the whole Wormix server stack and opens the game:
#
#   powershell -ExecutionPolicy Bypass -File .\start.ps1
#
# Each part runs in its own minimized window (restore one to see its log):
#   API :8000  ->  wormix-core :49197  ->  game host :32700
# Parts that are already running are left alone. Stop everything with stop.ps1.
# Needs only Go (winget install GoLang.Go); the three parts are built into .\bin\ on start.
#
# Options:
#   -NoBrowser    don't open the game in the browser
#   -Lan          let other computers (friends, Tailscale) open the game at http://<this PC>:32700
#   -GameLog      open the game with ?debug (game log lines appear in the game host window)

param(
    [switch]$NoBrowser,
    [switch]$Lan,
    [switch]$GameLog,
    # Game files (part of this repository).
    [string]$ClientDir = (Join-Path $PSScriptRoot 'wormix-client'),
    # The API's database and uploaded photos.
    [string]$DataDir   = (Join-Path $PSScriptRoot 'data')
)

$ErrorActionPreference = 'Stop'
$pidFile = Join-Path $PSScriptRoot '.pids'

function Step($text) { Write-Host "==> $text" -ForegroundColor Cyan }
function Fail($text) { Write-Host "ERROR: $text" -ForegroundColor Red; exit 1 }

function Test-Port([int]$port) {
    $c = New-Object Net.Sockets.TcpClient
    try { $c.ConnectAsync('127.0.0.1', $port).Wait(300) -and $c.Connected } catch { $false } finally { $c.Close() }
}

function Wait-Port([int]$port, [string]$name, [int]$seconds = 60) {
    $deadline = (Get-Date).AddSeconds($seconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-Port $port) { return }
        Start-Sleep -Milliseconds 500
    }
    Fail "$name did not start on port $port within $seconds s. Restore its window to see why."
}

function Find-Exe([string]$name, [string[]]$fallbacks) {
    $cmd = Get-Command $name -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    foreach ($f in $fallbacks) { $hit = Get-Item $f -ErrorAction SilentlyContinue | Select-Object -First 1; if ($hit) { return $hit.FullName } }
    return $null
}

function Start-Part([string]$name, [int]$port, [string]$exe, [string[]]$arguments, [string]$workDir) {
    if (Test-Port $port) {
        Write-Host "    $name is already running on port $port" -ForegroundColor DarkGray
        return
    }
    Step "Starting $name (port $port)"
    $opts = @{ FilePath = $exe; WorkingDirectory = $workDir; WindowStyle = 'Minimized'; PassThru = $true }
    if ($arguments.Count -gt 0) { $opts.ArgumentList = $arguments }
    $p = Start-Process @opts
    Add-Content -Path $pidFile -Value "$($p.Id) $name"
    Wait-Port $port $name
}

# --- Checks ---------------------------------------------------------------------------------
if (-not (Test-Path (Join-Path $ClientDir 'vk\1.25.0\preloader.swf'))) {
    Fail "Game client not found in $ClientDir (expected vk\1.25.0\preloader.swf). See README.md."
}
$go = Find-Exe 'go' @("$env:ProgramFiles\Go\bin\go.exe")
if (-not $go) { Fail 'Go not found. Install it with: winget install GoLang.Go' }

# --- Build (incremental, a few seconds after the first time) --------------------------------
$bin = Join-Path $PSScriptRoot 'bin'
foreach ($part in @(@('api', 8000), @('core', 49197), @('relay', 32700))) {
    $name, $port = $part
    if (Test-Port $port) { continue }
    Step "Building $name"
    $pkg = if ($name -eq 'core') { './cmd/wormix-core' } else { '.' }
    Push-Location (Join-Path $PSScriptRoot "services\$name")
    try { & $go build -o (Join-Path $bin "wormix-$name.exe") $pkg } finally { Pop-Location }
    if ($LASTEXITCODE -ne 0) { Fail "$name build failed (see the error above)." }
}

# --- Start ----------------------------------------------------------------------------------
$listen = if ($Lan) { '0.0.0.0' } else { '127.0.0.1' }
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

$env:DB_DATABASE = Join-Path $DataDir 'database.sqlite'
$env:LISTEN_ADDR = '127.0.0.1:8000'
Start-Part 'API' 8000 (Join-Path $bin 'wormix-api.exe') @() $DataDir

$coreConfig = Join-Path $PSScriptRoot 'configs\wormix-core.json'
Start-Part 'wormix-core' 49197 (Join-Path $bin 'wormix-core.exe') @("`"$coreConfig`"") $PSScriptRoot

$relayArgs = @('--Urls', "http://${listen}:32700", '--ClientRoot', "`"$ClientDir`"",
               '--Target', '127.0.0.1', '--ApiUrl', 'http://127.0.0.1:8000')
if ($GameLog) { $relayArgs += @('--DebugLog', 'true') }
Start-Part 'Game host' 32700 (Join-Path $bin 'wormix-relay.exe') $relayArgs $PSScriptRoot

# --- Done -----------------------------------------------------------------------------------
$url = 'http://127.0.0.1:32700/' + $(if ($GameLog) { '?debug' } else { '' })
Write-Host ''
Write-Host "Wormix is running: $url" -ForegroundColor Green
if ($Lan) {
    $ips = Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
        Where-Object { $_.IPAddress -notmatch '^(127\.|169\.254\.)' } | ForEach-Object { $_.IPAddress }
    foreach ($ip in $ips) { Write-Host "Friends can open:  http://${ip}:32700/" -ForegroundColor Green }
    Write-Host '(Windows may ask to allow the game host through the firewall: allow it for private networks.)'
}
Write-Host 'Stop everything with: powershell -ExecutionPolicy Bypass -File .\stop.ps1'
if (-not $NoBrowser) { Start-Process $url }
