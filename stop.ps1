# Stops everything start.ps1 started (each part and the programs it launched).
#   powershell -ExecutionPolicy Bypass -File .\stop.ps1

$pidFile = Join-Path $PSScriptRoot '.pids'
if (-not (Test-Path $pidFile)) {
    Write-Host 'Nothing to stop (no .pids file). Parts started by hand must be closed by hand.'
    exit 0
}

foreach ($line in Get-Content $pidFile) {
    $id, $name = $line -split ' ', 2
    if (Get-Process -Id $id -ErrorAction SilentlyContinue) {
        Write-Host "Stopping $name (pid $id)"
        & taskkill /PID $id /T /F | Out-Null
    }
}
Remove-Item $pidFile
Write-Host 'Stopped.' -ForegroundColor Green
