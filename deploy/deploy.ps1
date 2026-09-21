#Requires -Version 5.1
<#
.SYNOPSIS
  One-click Laya deploy on Windows (Go deploy server + model service).

.DESCRIPTION
  Builds laya-deploy / laya / laya-mcp, starts laya-deploy on the given port
  (default 7710), and waits until /health and /deploy/status are ok.

.EXAMPLE
  .\deploy\deploy.ps1
  .\deploy\deploy.ps1 -Port 7710 -SkipTests
#>
[CmdletBinding()]
param(
    [int]$Port = 7710,
    [string]$Bind = "0.0.0.0",
    [switch]$SkipTests,
    [switch]$Foreground
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

function Write-Step($msg) { Write-Host "[laya-deploy] $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[laya-deploy] $msg" -ForegroundColor Green }
function Write-Err($msg) { Write-Host "[laya-deploy] $msg" -ForegroundColor Red }

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    Write-Err "Go toolchain not found on PATH. Install Go 1.27+ first."
    exit 1
}
Write-Step "using $($go.Source)"

$binDir = Join-Path $RepoRoot "bin"
if (-not (Test-Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir | Out-Null
}

if (-not $SkipTests) {
    Write-Step "go test ./..."
    & go test ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Err "tests failed; abort deploy"
        exit 1
    }
}

Write-Step "building binaries"
& go build -o (Join-Path $binDir "laya-deploy.exe") .\deploy
if ($LASTEXITCODE -ne 0) { Write-Err "build laya-deploy failed"; exit 1 }
& go build -o (Join-Path $binDir "laya.exe") .\cli
if ($LASTEXITCODE -ne 0) { Write-Err "build laya failed"; exit 1 }
& go build -o (Join-Path $binDir "laya-mcp.exe") .\mcp
if ($LASTEXITCODE -ne 0) { Write-Err "build laya-mcp failed"; exit 1 }
Write-Ok "binaries ready in $binDir"

$addr = "{0}:{1}" -f $Bind, $Port
$url = "http://{0}" -f $addr
$exe = Join-Path $binDir "laya-deploy.exe"
$logPath = Join-Path $binDir "laya-deploy.log"
$errPath = Join-Path $binDir "laya-deploy.err.log"
$pidPath = Join-Path $binDir "laya-deploy.pid"
$healthUrl = "$url/health"
$statusUrl = "$url/deploy/status"

function Test-LayaUp {
    try {
        $h = Invoke-RestMethod -Uri $healthUrl -TimeoutSec 2
        $s = Invoke-RestMethod -Uri $statusUrl -TimeoutSec 2
        return ($h.status -eq "ok" -and $s.status -eq "ok" -and $s)
    } catch {
        return $null
    }
}

function Stop-LayaDeploy {
    Write-Step "stopping previous laya-deploy on $addr (if any)"
    if (Test-Path $pidPath) {
        $oldPid = 0
        try { $oldPid = [int](Get-Content $pidPath -Raw).Trim() } catch {}
        if ($oldPid -gt 0) {
            Stop-Process -Id $oldPid -Force -ErrorAction SilentlyContinue
        }
    }
    Get-Process -Name "laya-deploy" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    foreach ($conn in @($connections)) {
        $procId = $conn.OwningProcess
        try {
            $p = Get-Process -Id $procId -ErrorAction Stop
            if ($p.ProcessName -match "laya-deploy|laya-server") {
                Write-Step "stop pid=$procId name=$($p.ProcessName)"
                Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
            }
        } catch {}
    }
    Start-Sleep -Milliseconds 400
}

# Reuse a healthy instance on the same port instead of fighting it.
$existing = Test-LayaUp
if ($existing -and -not $Foreground) {
    Write-Ok "already healthy on $url (pid=$($existing.pid)); reusing"
    Write-Host ""
    Write-Host "  URL          : $url"
    Write-Host "  PID          : $($existing.pid)"
    Write-Host "  Health       : $healthUrl"
    Write-Host "  Deploy status: $statusUrl"
    Write-Host "  Agent env    : `$env:LAYA_URL = '$url'"
    exit 0
}

Stop-LayaDeploy

if ($Foreground) {
    Write-Ok "starting laya-deploy in foreground on $url"
    Write-Host "LAYA_URL=$url"
    & $exe -addr $addr
    exit $LASTEXITCODE
}

Write-Step "starting laya-deploy on $url"
# CREATE_NO_WINDOW — no stray console host on Windows.
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $exe
$psi.Arguments = "-addr $addr"
$psi.WorkingDirectory = "$RepoRoot"
$psi.UseShellExecute = $false
$psi.CreateNoWindow = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$proc = [System.Diagnostics.Process]::Start($psi)
$proc.Id | Out-File -FilePath $pidPath -Encoding ascii -Force

# Read redirected pipes on background threads so the child cannot block.
$logJob = Start-ThreadJob -ScriptBlock {
    param($proc, $logPath)
    $writer = [System.IO.StreamWriter]::new($logPath, $true)
    $writer.AutoFlush = $true
    try {
        while (-not $proc.HasExited) {
            $line = $proc.StandardOutput.ReadLine()
            if ($null -eq $line) { break }
            $writer.WriteLine($line)
        }
    } catch {} finally { $writer.Dispose() }
} -ArgumentList $proc, $logPath
$errJob = Start-ThreadJob -ScriptBlock {
    param($proc, $errPath)
    $writer = [System.IO.StreamWriter]::new($errPath, $true)
    $writer.AutoFlush = $true
    try {
        while (-not $proc.HasExited) {
            $line = $proc.StandardError.ReadLine()
            if ($null -eq $line) { break }
            $writer.WriteLine($line)
        }
    } catch {} finally { $writer.Dispose() }
} -ArgumentList $proc, $errPath

$deadline = (Get-Date).AddSeconds(15)
$ok = $null
while ((Get-Date) -lt $deadline) {
    if ($proc.HasExited) {
        Write-Err "laya-deploy exited early code=$($proc.ExitCode). See $logPath / $errPath"
        foreach ($f in @($logPath, $errPath)) {
            if (Test-Path $f) { Get-Content $f -Tail 40 }
        }
        exit 1
    }
    $ok = Test-LayaUp
    if ($ok) { break }
    Start-Sleep -Milliseconds 250
}

if (-not $ok) {
    Write-Err "health check timeout for $healthUrl"
    foreach ($f in @($logPath, $errPath)) {
        if (Test-Path $f) { Get-Content $f -Tail 40 }
    }
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    exit 1
}

Write-Ok "Laya deploy server is up"
Write-Host ""
Write-Host "  URL          : $url"
Write-Host "  PID          : $($proc.Id)"
Write-Host "  Health       : $healthUrl"
Write-Host "  Deploy status: $statusUrl"
Write-Host "  Models API   : $url/v1/models"
Write-Host "  Decide API   : $url/v1/decide"
Write-Host "  Log          : $logPath"
Write-Host ""
Write-Host "Agent env:"
Write-Host "  `$env:LAYA_URL = '$url'"
Write-Host ""
Write-Host "Try:"
Write-Host "  .\bin\laya.exe health --url $url"
Write-Host "  .\bin\laya.exe decide --url $url --feature intent_change=0.9 --feature target_known=0.8"
Write-Host ""
Write-Host "Stop:"
Write-Host "  Stop-Process -Id $($proc.Id) -Force"
Write-Host "  # or: Get-Process laya-deploy -ErrorAction SilentlyContinue | Stop-Process -Force"
