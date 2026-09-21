#Requires -Version 5.1
<#
.SYNOPSIS
  Remove global laya-mcp install (binary + PATH). Does not delete agent MCP entries.
#>
[CmdletBinding()]
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Laya\bin",
    [switch]$RemoveAgentEntries
)

$ErrorActionPreference = "Stop"
function Write-Step($msg) { Write-Host "[laya-mcp-uninstall] $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[laya-mcp-uninstall] $msg" -ForegroundColor Green }

Write-Step "remove binaries in $InstallDir"
if (Test-Path $InstallDir) {
    Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $InstallDir "laya-mcp.exe")
    Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $InstallDir "laya-mcp.cmd")
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath) {
    $parts = $userPath -split ';' | Where-Object { $_ -and ($_.TrimEnd('\') -ne $InstallDir.TrimEnd('\')) }
    [Environment]::SetEnvironmentVariable("Path", ($parts -join ';'), "User")
    Write-Ok "user PATH cleaned"
}

if ($RemoveAgentEntries) {
    $mimo = Join-Path $env:USERPROFILE ".config\mimocode\mimocode.jsonc"
    if (Test-Path $mimo) {
        try {
            $obj = Get-Content -Raw $mimo | ConvertFrom-Json
            if ($obj.mcp -and $obj.mcp.laya) {
                $obj.mcp.PSObject.Properties.Remove('laya')
                $obj | ConvertTo-Json -Depth 20 | Set-Content $mimo -Encoding utf8
                Write-Ok "removed mimocode laya entry"
            }
        } catch { Write-Host "mimocode cleanup skipped: $_" }
    }
    $codex = Join-Path $env:USERPROFILE ".codex\config.toml"
    if (Test-Path $codex) {
        $raw = Get-Content -Raw $codex
        $new = [regex]::Replace($raw, '(?ms)^\[mcp_servers\.laya\].*?(?=^\[|\z)', '')
        Set-Content -Path $codex -Value $new -Encoding utf8
        Write-Ok "removed codex laya block"
    }
}

Write-Ok "uninstall done"
