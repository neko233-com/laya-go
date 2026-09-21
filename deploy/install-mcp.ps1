#Requires -Version 5.1
<#
.SYNOPSIS
  One-click GLOBAL install for laya-mcp (binary on PATH + agent MCP configs).

.DESCRIPTION
  Builds/copies laya-mcp.exe into %LOCALAPPDATA%\Laya\bin, appends that dir to
  user PATH, then merges MCP registration into MiMo Desktop / Codex / Claude
  config files when present.

.EXAMPLE
  .\deploy\install-mcp.ps1
  .\deploy\install-mcp.ps1 -Url http://192.168.120.131:7710
  .\deploy\install-mcp.ps1 -SkipBuild -Agents mimocode,codex
#>
[CmdletBinding()]
param(
    [string]$Url = "",
    [string]$InstallDir = "$env:LOCALAPPDATA\Laya\bin",
    [string[]]$Agents = @("mimocode", "codex", "claude"),
    [switch]$SkipBuild,
    [switch]$SkipAgents
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

function Write-Step($msg) { Write-Host "[laya-mcp-install] $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[laya-mcp-install] $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "[laya-mcp-install] $msg" -ForegroundColor Yellow }
function Write-Err($msg) { Write-Host "[laya-mcp-install] $msg" -ForegroundColor Red }

function Test-LayaUrl([string]$u) {
    try {
        $h = Invoke-RestMethod -Uri "$u/health" -TimeoutSec 2
        return ($h.status -eq "ok")
    } catch { return $false }
}

function Get-JsonFile([string]$path) {
    if (-not (Test-Path $path)) { return $null }
    try {
        $raw = Get-Content -Raw -Path $path
        if ([string]::IsNullOrWhiteSpace($raw)) { return $null }
        # strip simple // line comments for jsonc
        $lines = $raw -split "`r?`n" | ForEach-Object {
            if ($_ -match '^\s*//') { "" } else { $_ }
        }
        $clean = ($lines -join "`n")
        return ($clean | ConvertFrom-Json)
    } catch {
        Write-Warn "parse failed: $path — $($_.Exception.Message)"
        return $null
    }
}

function Save-JsonFile([string]$path, $obj) {
    $dir = Split-Path -Parent $path
    if ($dir -and -not (Test-Path $dir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }
    $json = $obj | ConvertTo-Json -Depth 20
    Set-Content -Path $path -Value $json -Encoding utf8
}

# Resolve server URL
if (-not $Url) {
    if ($env:LAYA_URL) { $Url = $env:LAYA_URL }
    else { $Url = "http://127.0.0.1:7710" }
}
$Url = $Url.TrimEnd('/')

Write-Step "target LAYA_URL=$Url"
if (Test-LayaUrl $Url) {
    Write-Ok "laya server healthy at $Url"
} else {
    Write-Warn "laya server not reachable at $Url — install continues; fix server or pass -Url"
}

# Build / locate binary
$srcExe = Join-Path $RepoRoot "bin\laya-mcp.exe"
if (-not $SkipBuild) {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) {
        Write-Step "go build ./mcp"
        if (-not (Test-Path (Join-Path $RepoRoot "bin"))) {
            New-Item -ItemType Directory -Path (Join-Path $RepoRoot "bin") | Out-Null
        }
        & go build -o $srcExe .\mcp
        if ($LASTEXITCODE -ne 0) { Write-Err "go build failed"; exit 1 }
    } elseif (Test-Path $srcExe) {
        Write-Step "Go not found; using existing $srcExe"
    } else {
        # try ProgramData service install copy
        $pd = Join-Path $env:ProgramData "Laya\laya-mcp.exe"
        if (Test-Path $pd) {
            Write-Step "using $pd"
            $srcExe = $pd
        } else {
            Write-Err "laya-mcp.exe not found. Install Go or run .\deploy\install-service.ps1 first."
            exit 1
        }
    }
} elseif (-not (Test-Path $srcExe)) {
    $pd = Join-Path $env:ProgramData "Laya\laya-mcp.exe"
    if (Test-Path $pd) { $srcExe = $pd }
}

if (-not (Test-Path $srcExe)) {
    Write-Err "source binary missing: $srcExe"
    exit 1
}

Write-Step "install to $InstallDir"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$destExe = Join-Path $InstallDir "laya-mcp.exe"
Copy-Item -Force $srcExe $destExe

# Also expose a .cmd shim for PATH convenience
$shim = Join-Path $InstallDir "laya-mcp.cmd"
@"
@echo off
setlocal
if "%LAYA_URL%"=="" set "LAYA_URL=$Url"
"%~dp0laya-mcp.exe" %*
"@ | Set-Content -Path $shim -Encoding ascii

# User PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$parts = @()
if ($userPath) { $parts = $userPath -split ';' | Where-Object { $_ -and $_.Trim() } }
if ($parts -notcontains $InstallDir) {
    $parts += $InstallDir
    $newPath = ($parts -join ';').TrimEnd(';')
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Ok "user PATH += $InstallDir (new shells pick this up)"
} else {
    Write-Ok "user PATH already contains $InstallDir"
}
$env:Path = "$InstallDir;$env:Path"

$mcpExe = $destExe
$mcpArgsNote = "stdio MCP server"

function Upsert-MiMo([string]$url) {
    $path = Join-Path $env:USERPROFILE ".config\mimocode\mimocode.jsonc"
    if (-not (Test-Path $path)) {
        $path = Join-Path $env:USERPROFILE ".config\mimocode\mimocode.json"
    }
    if (-not (Test-Path $path)) {
        Write-Warn "MiMo config not found; skip (path tried under %USERPROFILE%\.config\mimocode)"
        return
    }
    $obj = Get-JsonFile $path
    if ($null -eq $obj) { return }
    if (-not $obj.mcp) {
        $obj | Add-Member -NotePropertyName mcp -NotePropertyValue ([pscustomobject]@{}) -Force
    }
    $entry = [pscustomobject]@{
        type        = "local"
        command     = @($mcpExe)
        environment = [pscustomobject]@{ LAYA_URL = $url }
        enabled     = $true
    }
    $obj.mcp | Add-Member -NotePropertyName laya -NotePropertyValue $entry -Force
    Save-JsonFile $path $obj
    Write-Ok "MiMo Desktop MCP -> $path"
}

function Upsert-Codex([string]$url) {
    $path = Join-Path $env:USERPROFILE ".codex\config.toml"
    $dir = Split-Path -Parent $path
    if (-not (Test-Path $dir)) {
        Write-Warn "Codex config dir missing; skip $path"
        return
    }
    # TOML literal string: single quotes, no backslash escaping
    $cmdLit = $mcpExe -replace "'", "''"
    $block = @"

[mcp_servers.laya]
command = '$cmdLit'
env = { LAYA_URL = "$url" }
"@
    if (-not (Test-Path $path)) {
        Set-Content -Path $path -Value $block.TrimStart() -Encoding utf8
        Write-Ok "Codex MCP created $path"
        return
    }
    $raw = Get-Content -Raw -Path $path
    if ($raw -match '(?m)^\[mcp_servers\.laya\]') {
        $pattern = '(?ms)^\[mcp_servers\.laya\].*?(?=^\[|\z)'
        $new = [regex]::Replace($raw, $pattern, ($block.TrimStart() + "`n"))
        Set-Content -Path $path -Value $new -Encoding utf8
        Write-Ok "Codex MCP updated $path"
    } else {
        if ($raw -and -not $raw.EndsWith("`n")) { $raw += "`n" }
        Set-Content -Path $path -Value ($raw + $block.TrimStart() + "`n") -Encoding utf8
        Write-Ok "Codex MCP appended $path"
    }
}

function Upsert-Claude([string]$url) {
    $candidates = @(
        (Join-Path $env:APPDATA "Claude\claude_desktop_config.json"),
        (Join-Path $env:USERPROFILE ".claude.json")
    )
    $updated = $false
    foreach ($path in $candidates) {
        if (-not (Test-Path $path)) { continue }
        $obj = Get-JsonFile $path
        if ($null -eq $obj) { continue }
        if (-not $obj.mcpServers) {
            $obj | Add-Member -NotePropertyName mcpServers -NotePropertyValue ([pscustomobject]@{}) -Force
        }
        $entry = [pscustomobject]@{
            command = $mcpExe
            env     = [pscustomobject]@{ LAYA_URL = $url }
        }
        $obj.mcpServers | Add-Member -NotePropertyName laya -NotePropertyValue $entry -Force
        Save-JsonFile $path $obj
        Write-Ok "Claude MCP -> $path"
        $updated = $true
    }
    if (-not $updated) {
        Write-Warn "Claude config not found; skip"
    }
}

if (-not $SkipAgents) {
    foreach ($a in $Agents) {
        switch ($a.ToLower()) {
            "mimocode" { Upsert-MiMo $Url }
            "mimo" { Upsert-MiMo $Url }
            "codex" { Upsert-Codex $Url }
            "claude" { Upsert-Claude $Url }
            default { Write-Warn "unknown agent $a" }
        }
    }
}

# Verify stdio against server
Write-Step "verify laya-mcp stdio"
$payload = @(
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"install-mcp","version":"1"}}}'
    '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
) -join "`n"
$env:LAYA_URL = $Url
$resp = $payload | & $destExe
$okInit = ($resp -match 'laya-mcp')
$okTools = ($resp -match 'laya_decide')
if ($okInit -and $okTools) {
    Write-Ok "laya-mcp stdio OK (initialize + tools/list)"
} else {
    Write-Warn "stdio verify incomplete; run manually:"
    Write-Host "  `$env:LAYA_URL='$Url'; echo '{`"jsonrpc`":`"2.0`",`"id`":1,`"method`":`"tools/list`"}' | & '$destExe'"
}

Write-Ok "GLOBAL install done"
Write-Host ""
Write-Host "  Binary : $destExe"
Write-Host "  Shim   : $shim"
Write-Host "  LAYA_URL default: $Url"
Write-Host "  MCP name in agents: laya"
Write-Host "  Tools  : laya_health / laya_models / laya_decide"
Write-Host ""
Write-Host "Agent hosts: restart Codex / MiMo / Claude after install so MCP loads."
Write-Host "Docs: docs/agent-integration.md"
Write-Host ""
Write-Host "Uninstall PATH entry manually if needed, or delete: $InstallDir"
