#Requires -Version 5.1
<#
.SYNOPSIS
  Stop and remove the LayaDeploy Windows service.

.EXAMPLE
  .\deploy\uninstall-service.ps1
  .\deploy\uninstall-service.ps1 -RemoveFiles
#>
[CmdletBinding()]
param(
    [string]$ServiceName = "LayaDeploy",
    [string]$InstallDir = "$env:ProgramData\Laya",
    [switch]$RemoveFiles
)

$ErrorActionPreference = "Stop"
function Write-Step($msg) { Write-Host "[laya-service] $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[laya-service] $msg" -ForegroundColor Green }
function Write-Err($msg) { Write-Host "[laya-service] $msg" -ForegroundColor Red }

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Err "Administrator rights are required. Re-run elevated."
    exit 1
}

$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if (-not $svc) {
    Write-Ok "service $ServiceName not installed"
} else {
    Write-Step "stopping $ServiceName"
    if ($svc.Status -ne "Stopped") {
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 1
    }
    Write-Step "deleting $ServiceName"
    & sc.exe delete $ServiceName | Out-Null
}

Get-Process -Name "laya-deploy" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

if ($RemoveFiles) {
    Write-Step "removing $InstallDir"
    if (Test-Path $InstallDir) {
        Remove-Item -Recurse -Force $InstallDir
    }
}

Write-Ok "uninstall done"
