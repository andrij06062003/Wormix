# Starts the whole Wormix server stack and opens the game:
#
#   powershell -ExecutionPolicy Bypass -File .\start.ps1
#
# Each part runs in its own minimized window (restore one to see its log):
#   Redis (Garnet) :6379  ->  API :8000  ->  wormix-core :49197  ->  game host :32700
# Parts that are already running are left alone. Stop everything with stop.ps1.
#
# Options:
#   -NoBrowser    don't open the game in the browser
#   -Lan          let other computers (friends, Tailscale) open the game at http://<this PC>:32700
#   -GameLog      open the game with ?debug (game log lines appear in the game host window)

param(
    [switch]$NoBrowser,
    [switch]$Lan,
    [switch]$GameLog,
    # Sibling folders, by default next to this one.
    [string]$ApiDir    = (Join-Path (Split-Path $PSScriptRoot) 'wormix-api'),
    [string]$CoreDir   = (Join-Path (Split-Path $PSScriptRoot) 'wormix-core'),
    [string]$ClientDir = (Join-Path (Split-Path $PSScriptRoot) 'wormix-client')
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
    $p = Start-Process -FilePath $exe -ArgumentList $arguments -WorkingDirectory $workDir -WindowStyle Minimized -PassThru
    Add-Content -Path $pidFile -Value "$($p.Id) $name"
    Wait-Port $port $name
}

# --- Checks ---------------------------------------------------------------------------------
foreach ($d in @($ApiDir, $CoreDir, $ClientDir)) {
    if (-not (Test-Path $d)) { Fail "Folder not found: $d" }
}
if (-not (Test-Path (Join-Path $ClientDir 'vk\1.05.0\preloader.swf'))) {
    Fail "Game client not found in $ClientDir (expected vk\1.05.0\preloader.swf). See README.md."
}
if (-not (Test-Path (Join-Path $ApiDir 'database\database.sqlite'))) {
    Fail "API is not set up yet. Run: powershell -ExecutionPolicy Bypass -File `"$ApiDir\setup.ps1`" -NoServe"
}

$php = Find-Exe 'php' @("$env:LOCALAPPDATA\Microsoft\WinGet\Packages\PHP.PHP.8.*\php.exe")
if (-not $php) { Fail 'PHP not found. Install it with: winget install PHP.PHP.8.3' }
$dotnet = Find-Exe 'dotnet' @("$env:ProgramFiles\dotnet\dotnet.exe")
if (-not $dotnet) { Fail '.NET not found. Install it with: winget install Microsoft.DotNet.SDK.8' }
$garnet = Find-Exe 'garnet-server' @("$env:USERPROFILE\.dotnet\tools\garnet-server.exe")
if (-not $garnet) { Fail 'Garnet (Redis) not found. Install it with: dotnet tool install --global garnet-server' }

# --- Build (incremental, a few seconds) ------------------------------------------------------
$coreProject  = Join-Path $CoreDir 'wormix-core\wormix-core.csproj'
$relayProject = Join-Path $PSScriptRoot 'wormix-relay.csproj'
if (-not (Test-Port 49197)) {
    Step 'Building wormix-core'
    & $dotnet build $coreProject -c Release -nologo -v q | Out-Null
    if ($LASTEXITCODE -ne 0) { Fail 'wormix-core build failed. Run the build yourself to see the error.' }
}
if (-not (Test-Port 32700)) {
    Step 'Building game host'
    & $dotnet build $relayProject -c Release -nologo -v q | Out-Null
    if ($LASTEXITCODE -ne 0) { Fail 'Game host build failed. Run the build yourself to see the error.' }
}

# --- Start ----------------------------------------------------------------------------------
$listen = if ($Lan) { '0.0.0.0' } else { '127.0.0.1' }

Start-Part 'Redis (Garnet)' 6379 $garnet @('--bind', '127.0.0.1', '--port', '6379') $PSScriptRoot

Start-Part 'API' 8000 $php @('artisan', 'serve', '--host=127.0.0.1', '--port=8000') $ApiDir

$coreDll = Join-Path $CoreDir 'wormix-core\bin\Release\net8.0\wormix-core.dll'
$coreConfig = Join-Path $PSScriptRoot 'configs\wormix-core.json'
Start-Part 'wormix-core' 49197 $dotnet @("`"$coreDll`"", "`"$coreConfig`"") $CoreDir

$relayDll = Join-Path $PSScriptRoot 'bin\Release\net8.0\wormix-relay.dll'
$relayArgs = @("`"$relayDll`"", '--Urls', "http://${listen}:32700", '--ClientRoot', "`"$ClientDir`"")
if ($GameLog) { $relayArgs += @('--DebugLog', 'true') }
Start-Part 'Game host' 32700 $dotnet $relayArgs $PSScriptRoot

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
