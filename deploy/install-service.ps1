#Requires -Version 5.1
<#
.SYNOPSIS
  Install laya-deploy as a Windows service (default Port 7710) with optional auto-update.

.EXAMPLE
  .\deploy\install-service.ps1
  .\deploy\install-service.ps1 -Port 7710 -AutoUpdate:$false
  .\deploy\install-service.ps1 -AutoApply:$false -IntervalMinutes 720
#>
[CmdletBinding()]
param(
    [int]$Port = 7710,
    [string]$Bind = "0.0.0.0",
    [string]$ServiceName = "LayaDeploy",
    [string]$InstallDir = "$env:ProgramData\Laya",
    [string]$Repo = "neko233-com/laya-go",
    [bool]$AutoUpdate = $true,
    [bool]$AutoApply = $true,
    [int]$IntervalMinutes = 360,
    [switch]$SkipTests,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

function Write-Step($msg) { Write-Host "[laya-service] $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[laya-service] $msg" -ForegroundColor Green }
function Write-Err($msg) { Write-Host "[laya-service] $msg" -ForegroundColor Red }

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Err "Administrator rights are required to install a Windows service."
    Write-Err "Re-run PowerShell as Administrator:"
    Write-Err "  Set-Location '$RepoRoot'; .\deploy\install-service.ps1 -Port $Port"
    exit 1
}

$go = Get-Command go -ErrorAction SilentlyContinue
$binDir = Join-Path $RepoRoot "bin"

if (-not $SkipBuild) {
    if (-not $go) {
        Write-Err "Go toolchain not found on PATH."
        exit 1
    }
    if (-not $SkipTests) {
        Write-Step "go test ./..."
        & go test ./...
        if ($LASTEXITCODE -ne 0) { Write-Err "tests failed"; exit 1 }
    }
    Write-Step "building binaries"
    if (-not (Test-Path $binDir)) { New-Item -ItemType Directory -Path $binDir | Out-Null }
    & go build -o (Join-Path $binDir "laya-deploy.exe") .\deploy
    if ($LASTEXITCODE -ne 0) { Write-Err "build laya-deploy failed"; exit 1 }
    & go build -o (Join-Path $binDir "laya.exe") .\cli
    if ($LASTEXITCODE -ne 0) { Write-Err "build laya failed"; exit 1 }
    & go build -o (Join-Path $binDir "laya-mcp.exe") .\mcp
    if ($LASTEXITCODE -ne 0) { Write-Err "build laya-mcp failed"; exit 1 }
}

foreach ($n in @("laya-deploy.exe", "laya.exe", "laya-mcp.exe")) {
    if (-not (Test-Path (Join-Path $binDir $n))) {
        Write-Err "missing binary $n in $binDir"
        exit 1
    }
}

Write-Step "install dir: $InstallDir"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$configPath = Join-Path $InstallDir "config.json"
$addr = "{0}:{1}" -f $Bind, $Port
$url = "http://{0}" -f $addr

$config = [ordered]@{
    addr = $addr
    auto_update = [ordered]@{
        enabled           = [bool]$AutoUpdate
        auto_apply        = [bool]$AutoApply
        interval_minutes  = $IntervalMinutes
        repo              = $Repo
        prerelease        = $false
    }
}
$config | ConvertTo-Json -Depth 5 | Set-Content -Path $configPath -Encoding UTF8
Write-Ok "wrote $configPath"
Write-Host ($config | ConvertTo-Json -Depth 5)

# Stop existing service / stray process on port.
Write-Step "stopping previous service/process if any"
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -ne "Stopped") {
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 1
    }
}
Get-Process -Name "laya-deploy" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 400

Write-Step "copying binaries to $InstallDir"
Copy-Item -Force (Join-Path $binDir "laya-deploy.exe") (Join-Path $InstallDir "laya-deploy.exe")
Copy-Item -Force (Join-Path $binDir "laya.exe") (Join-Path $InstallDir "laya.exe")
Copy-Item -Force (Join-Path $binDir "laya-mcp.exe") (Join-Path $InstallDir "laya-mcp.exe")

$exePath = Join-Path $InstallDir "laya-deploy.exe"
$binPath = '"{0}" -service -config "{1}" -addr {2}' -f $exePath, $configPath, $addr
$display = "Laya Deploy Server"
$desc = "Laya System 1 decision model service (HTTP on $addr) with optional auto-update"

Write-Step "registering service $ServiceName"
if ($svc) {
    & sc.exe stop $ServiceName | Out-Null
    Start-Sleep -Milliseconds 500
    & sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 1
}

& sc.exe create $ServiceName binPath= $binPath start= auto DisplayName= $display
if ($LASTEXITCODE -ne 0) {
    Write-Err "sc create failed"
    exit 1
}
& sc.exe description $ServiceName $desc | Out-Null
& sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/10000/""/0 | Out-Null

Write-Step "starting $ServiceName"
Start-Service -Name $ServiceName
Start-Sleep -Seconds 1

# Allow LAN inbound on the listen port (Windows Firewall).
if ($Bind -eq "0.0.0.0" -or $Bind -eq "+") {
    Write-Step "ensuring firewall allow rule for TCP $Port"
    $ruleName = "LayaDeploy-TCP-$Port"
    $existing = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if (-not $existing) {
        New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Action Allow -Protocol TCP -LocalPort $Port -Profile Any -ErrorAction SilentlyContinue | Out-Null
        if ($LASTEXITCODE -ne 0 -and -not (Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue)) {
            & netsh advfirewall firewall add rule name="$ruleName" dir=in action=allow protocol=TCP localport=$Port | Out-Null
        }
    }
}

$lanIPs = @(Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
    Where-Object { $_.IPAddress -notlike "127.*" -and $_.IPAddress -notlike "169.254.*" } |
    Select-Object -ExpandProperty IPAddress -Unique)
$listenUrl = if ($Bind -eq "0.0.0.0") { "http://127.0.0.1:$Port" } else { $url }

$healthUrl = "$url/health"
$statusUrl = "$url/deploy/status"
$configApi = "$url/deploy/config"
$updateCheck = "$url/deploy/update/check"
$deadline = (Get-Date).AddSeconds(20)
$ok = $null
while ((Get-Date) -lt $deadline) {
    try {
        $h = Invoke-RestMethod -Uri $healthUrl -TimeoutSec 2
        $s = Invoke-RestMethod -Uri $statusUrl -TimeoutSec 2
        if ($h.status -eq "ok" -and $s.status -eq "ok") { $ok = $s; break }
    } catch { Start-Sleep -Milliseconds 300 }
}

if (-not $ok) {
    Write-Err "service health check failed. Query events / status:"
    Write-Err "  Get-Service $ServiceName | Format-List *"
    Write-Err "  Get-WinEvent -LogName Application -MaxLines 20 | Where-Object { `$_.Message -match 'laya' }"
    & sc.exe query $ServiceName
    exit 1
}

Write-Ok "Windows service installed and healthy"
Write-Host ""
Write-Host "  Service      : $ServiceName"
Write-Host "  Display      : $display"
Write-Host "  Install dir  : $InstallDir"
Write-Host "  Executable   : $exePath"
Write-Host "  Config       : $configPath"
Write-Host "  Listen       : $addr"
Write-Host "  Local URL    : $listenUrl"
Write-Host "  LAN URLs     : $(if ($lanIPs) { ($lanIPs | ForEach-Object { "http://{0}:$Port" -f $_ }) -join ', ' } else { '(none detected)' })"
Write-Host "  PID          : $($ok.pid)"
Write-Host "  Version      : $($ok.version)"
Write-Host "  Auto update  : enabled=$AutoUpdate auto_apply=$AutoApply interval_min=$IntervalMinutes"
Write-Host ""
Write-Host "  Health       : $healthUrl"
Write-Host "  Status       : $statusUrl"
Write-Host "  Config API   : $configApi"
Write-Host "  Update check : $updateCheck"
Write-Host ""
Write-Host "Agent env (local):"
Write-Host "  `$env:LAYA_URL = '$listenUrl'"
Write-Host "Agent env (LAN peers):"
Write-Host "  `$env:LAYA_URL = 'http://<this-host-ip>:$Port'"
Write-Host "  `$env:LAYA_CONFIG = '$configPath'"
Write-Host ""
Write-Host "WARNING: bind $Bind + no auth = LAN-open decision API. Firewall/trust boundary is yours."
Write-Host ""
Write-Host "Toggle auto-update (no restart needed for the flag):"
Write-Host "  Invoke-RestMethod -Method Post $configApi -ContentType application/json -Body '{\"auto_update\":{\"enabled\":false,\"auto_apply\":false,\"interval_minutes\":360,\"repo\":\"$Repo\",\"prerelease\":false}}'"
Write-Host ""
Write-Host "Uninstall:"
Write-Host "  .\deploy\uninstall-service.ps1"
