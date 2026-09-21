#Requires -Version 5.1
<#
.SYNOPSIS
  Build release binaries and publish a GitHub release for laya auto-update.

.DESCRIPTION
  Creates platform assets named:
    laya-deploy_windows_amd64.exe
    laya_windows_amd64.exe
    laya-mcp_windows_amd64.exe

.EXAMPLE
  .\deploy\publish-release.ps1 -Tag v0.2.0
  .\deploy\publish-release.ps1 -Tag v0.2.0 -Prerelease
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Tag,
    [string]$Repo = "neko233-com/laya-go",
    [string]$Goos = "windows",
    [string]$Goarch = "amd64",
    [switch]$Prerelease,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

function Write-Step($msg) { Write-Host "[laya-release] $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[laya-release] $msg" -ForegroundColor Green }

$tagNorm = if ($Tag.StartsWith("v")) { $Tag } else { "v$Tag" }
$ver = $tagNorm.TrimStart("v")

# Keep internal/version in sync when publishing.
$versionFile = Join-Path $RepoRoot "internal\version\version.go"
if (Test-Path $versionFile) {
    $content = Get-Content $versionFile -Raw
    $content = $content -replace 'const Version = "[^"]+"', ('const Version = "{0}"' -f $ver)
    Set-Content -Path $versionFile -Value $content -Encoding UTF8
    Write-Step "internal/version set to $ver"
}

if (-not $SkipTests) {
    Write-Step "go test ./..."
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw "tests failed" }
}

$dist = Join-Path $RepoRoot "dist"
if (Test-Path $dist) { Remove-Item -Recurse -Force $dist }
New-Item -ItemType Directory -Path $dist | Out-Null

$env:GOOS = $Goos
$env:GOARCH = $Goarch
$env:CGO_ENABLED = "0"

$targets = @(
    @{ pkg = "./deploy"; out = "laya-deploy_${Goos}_${Goarch}.exe" },
    @{ pkg = "./cli";    out = "laya_${Goos}_${Goarch}.exe" },
    @{ pkg = "./mcp";    out = "laya-mcp_${Goos}_${Goarch}.exe" }
)

if ($Goos -ne "windows") {
    foreach ($t in $targets) { $t["out"] = $t["out"] -replace '\.exe$', '' }
}

foreach ($t in $targets) {
    $outName = $t["out"]
    $outPath = Join-Path $dist $outName
    Write-Step "build $($t["pkg"]) -> $outName"
    & go build -trimpath -ldflags "-s -w" -o $outPath $t["pkg"]
    if ($LASTEXITCODE -ne 0) { throw "build $($t["pkg"]) failed" }
}

Write-Ok "assets ready:"
Get-ChildItem $dist | ForEach-Object { Write-Host "  $($_.Name) $($_.Length)" }

Write-Step "checking git tag $tagNorm"
git fetch --tags origin 2>$null
$existing = git tag -l $tagNorm
if (-not $existing) {
    git -c user.name="neko233-com" -c user.email="neko233-com@users.noreply.github.com" tag $tagNorm
    git push origin $tagNorm
} else {
    Write-Step "tag $tagNorm already exists"
}

Write-Step "gh release create $tagNorm ($Repo)"
$notes = "Laya System 1 deploy release $tagNorm"
$prereleaseFlag = @()
if ($Prerelease) { $prereleaseFlag = @("--prerelease") }

$assets = Get-ChildItem $dist | ForEach-Object { $_.FullName }
# Replace existing release if present.
& gh release view $tagNorm --repo $Repo 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Step "uploading assets to existing release"
    & gh release upload $tagNorm --repo $Repo --clobber @assets
} else {
    & gh release create $tagNorm --repo $Repo --title $tagNorm --notes $notes @prereleaseFlag @assets
}
if ($LASTEXITCODE -ne 0) { throw "gh release operation failed" }

Write-Ok "published $tagNorm to https://github.com/$Repo/releases/tag/$tagNorm"
Write-Host "Windows services will pick it up on the next auto-update check (or POST /deploy/update/apply)."
