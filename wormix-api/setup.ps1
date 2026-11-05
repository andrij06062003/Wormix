# Sets up and starts the Wormix API on Windows with one command:
#   powershell -ExecutionPolicy Bypass -File .\setup.ps1
#
# Safe to run again: it skips steps that are already done and then starts the server.
# Options:
#   -NoServe   set everything up but don't start the server
#   -Port 8000 port for the server (wormix-web expects 8000)

param(
    [switch]$NoServe,
    [int]$Port = 8000
)

$ErrorActionPreference = 'Stop'
Set-Location -Path $PSScriptRoot

function Step($text) { Write-Host "`n==> $text" -ForegroundColor Cyan }
function Fail($text) { Write-Host "`nERROR: $text" -ForegroundColor Red; exit 1 }
function Run($exe, [string[]]$arguments) {
    & $exe @arguments
    if ($LASTEXITCODE -ne 0) { Fail "'$exe $($arguments -join ' ')' failed (exit code $LASTEXITCODE)." }
}

# 1. Required tools
Step 'Checking PHP and Composer'
if (-not (Get-Command php -ErrorAction SilentlyContinue)) {
    Fail "PHP is not installed or not on PATH. Install PHP 8.2+ (for example from https://windows.php.net/download or with Laragon), then open a new terminal."
}
$phpVersion = (& php -r 'echo PHP_VERSION;')
if ([version]($phpVersion -replace '[^0-9.].*$', '') -lt [version]'8.2') {
    Fail "PHP $phpVersion found, but 8.2 or newer is required."
}
Write-Host "PHP $phpVersion"

$extensions = (& php -m) -join "`n"
$missing = @('pdo_sqlite', 'openssl', 'simplexml', 'mbstring', 'fileinfo') |
    Where-Object { $extensions -notmatch "(?im)^$_$" }
if ($missing) {
    & php --ini
    Fail ("PHP extensions missing: $($missing -join ', '). " +
          "Enable them in the php.ini shown above (remove the ';' in front of 'extension=<name>'). " +
          "If no php.ini is loaded, copy php.ini-development to php.ini in the PHP folder first. Then run this script again.")
}

if (-not (Get-Command composer -ErrorAction SilentlyContinue)) {
    Fail 'Composer is not installed or not on PATH. Install it from https://getcomposer.org/download/, then open a new terminal.'
}

# 2. Config file
Step 'Preparing .env'
if (-not (Test-Path .env)) {
    Copy-Item .env.example .env
    Write-Host 'Created .env from .env.example'
} else {
    Write-Host '.env already exists, keeping it'
}

# 3. PHP packages
Step 'Installing PHP packages (composer install)'
Run composer @('install', '--no-interaction')

# 4. App key
$envText = Get-Content .env -Raw
if ($envText -match '(?m)^APP_KEY=\s*$') {
    Step 'Generating app key'
    Run php @('artisan', 'key:generate', '--force')
}

# 5. Database
$usesSqlite = $envText -match '(?m)^DB_CONNECTION=sqlite\s*$'
$freshDatabase = $false
if ($usesSqlite) {
    $dbFile = Join-Path $PSScriptRoot 'database\database.sqlite'
    if (-not (Test-Path $dbFile)) {
        New-Item -ItemType File -Path $dbFile | Out-Null
        $freshDatabase = $true
        Write-Host "Created SQLite database: database\database.sqlite"
    }
}

Step 'Creating or updating database tables'
Run php @('artisan', 'migrate', '--force')

# 6. Game data (weapons, hats, missions...). Only on a new database: running it twice reports duplicates.
if ($freshDatabase -or -not $usesSqlite) {
    if ($freshDatabase) {
        Step 'Loading game data (game:init)'
        Run php @('artisan', 'game:init')
    } else {
        Write-Host "`nUsing a non-SQLite database: run 'php artisan game:init' yourself once, on an empty database." -ForegroundColor Yellow
    }
} else {
    Write-Host "`nGame data was loaded earlier, skipping game:init. To start over, delete database\database.sqlite and run this script again."
}

# 7. Start
if ($NoServe) {
    Write-Host "`nSetup complete. Start the server with: php artisan serve --port=$Port" -ForegroundColor Green
    exit 0
}
Step "Starting the API at http://127.0.0.1:$Port/api/  (Ctrl+C to stop)"
Run php @('artisan', 'serve', "--port=$Port")
