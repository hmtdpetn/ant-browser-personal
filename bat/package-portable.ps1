param(
    [string]$Version = "1.3.0-personal.1",
    [switch]$SkipBuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$publishScript = Join-Path $PSScriptRoot "publish.ps1"

if (-not (Test-Path -LiteralPath $publishScript -PathType Leaf)) {
    throw "Missing bat\publish.ps1"
}

if ($SkipBuild) {
    Write-Warning "V1.3 portable packaging always performs a verified build; -SkipBuild is ignored."
}

Push-Location $repoRoot
try {
    & $publishScript -Target WINDOWS -WindowsFormat PORTABLE -Version $Version
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
