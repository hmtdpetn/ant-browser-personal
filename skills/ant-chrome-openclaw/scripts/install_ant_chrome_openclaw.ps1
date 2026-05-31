<#
.SYNOPSIS
Install the ant-chrome-openclaw skill into an OpenClaw skills directory and optionally merge openclaw.json.

.EXAMPLE
pwsh -File install_ant_chrome_openclaw.ps1 -TargetSkillsDir "C:\OpenClaw\skills" -ConfigFile "C:\OpenClaw\openclaw.json" -SetDefaultProfile

.EXAMPLE
pwsh -File install_ant_chrome_openclaw.ps1 -TargetSkillsDir "C:\OpenClaw\skills" -DryRun
#>

param(
    [string]$TargetSkillsDir = "",
    [string]$ConfigFile = "",
    [string]$BrowserProfileName = "ant-chrome",
    [string]$BaseUrl = "",
    [string]$ApiHeader = "",
    [string]$ApiKey = "",
    [string]$Color = "#0F766E",
    [switch]$Help,
    [switch]$SetDefaultProfile,
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-DefaultValue {
    param(
        [string]$Value,
        [string]$Fallback
    )

    if ([string]::IsNullOrWhiteSpace($Value)) {
        return $Fallback
    }
    return $Value.Trim()
}

function Add-Candidate {
    param(
        [System.Collections.Generic.List[string]]$List,
        [string]$Value
    )

    if (-not [string]::IsNullOrWhiteSpace($Value)) {
        $List.Add($Value.Trim())
    }
}

function Find-ExistingPath {
    param(
        [string[]]$Candidates
    )

    foreach ($candidate in $Candidates) {
        if ([string]::IsNullOrWhiteSpace($candidate)) {
            continue
        }
        if (Test-Path -LiteralPath $candidate) {
            return (Resolve-Path -LiteralPath $candidate).Path
        }
    }
    return ""
}

function ConvertTo-PlainData {
    param(
        [Parameter(ValueFromPipeline = $true)]
        $Value
    )

    if ($null -eq $Value) {
        return $null
    }

    if ($Value -is [System.Collections.IDictionary]) {
        $map = [ordered]@{}
        foreach ($key in $Value.Keys) {
            $map[[string]$key] = ConvertTo-PlainData $Value[$key]
        }
        return $map
    }

    if ($Value -is [pscustomobject]) {
        $map = [ordered]@{}
        foreach ($prop in $Value.PSObject.Properties) {
            $map[$prop.Name] = ConvertTo-PlainData $prop.Value
        }
        return $map
    }

    if (($Value -is [System.Collections.IEnumerable]) -and -not ($Value -is [string])) {
        $items = @()
        foreach ($item in $Value) {
            $items += ,(ConvertTo-PlainData $item)
        }
        return $items
    }

    return $Value
}

function Ensure-MapNode {
    param(
        [System.Collections.IDictionary]$Parent,
        [string]$Key
    )

    $existing = $Parent[$Key]
    if (-not ($existing -is [System.Collections.IDictionary])) {
        $Parent[$Key] = [ordered]@{}
    }
    return $Parent[$Key]
}

function Read-JsonFile {
    param(
        [string]$Path
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return [ordered]@{}
    }

    $raw = [System.IO.File]::ReadAllText((Resolve-Path -LiteralPath $Path))
    if ([string]::IsNullOrWhiteSpace($raw)) {
        return [ordered]@{}
    }

    return ConvertTo-PlainData ($raw | ConvertFrom-Json)
}

function Write-JsonFile {
    param(
        [string]$Path,
        [System.Collections.IDictionary]$Data
    )

    $parent = Split-Path -Parent $Path
    if (-not [string]::IsNullOrWhiteSpace($parent)) {
        [System.IO.Directory]::CreateDirectory($parent) | Out-Null
    }

    $json = $Data | ConvertTo-Json -Depth 50
    $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
    [System.IO.File]::WriteAllText($Path, $json + [Environment]::NewLine, $utf8NoBom)
}

function Get-SourceSkillRoot {
    $scriptDir = Split-Path -Parent $PSCommandPath
    return Split-Path -Parent $scriptDir
}

function Get-DefaultSkillsDir {
    $candidates = [System.Collections.Generic.List[string]]::new()
    Add-Candidate $candidates $env:OPENCLAW_SKILLS_DIR
    if (-not [string]::IsNullOrWhiteSpace($env:OPENCLAW_HOME)) {
        Add-Candidate $candidates (Join-Path $env:OPENCLAW_HOME "skills")
    }
    Add-Candidate $candidates (Join-Path $HOME ".openclaw\skills")
    if (-not [string]::IsNullOrWhiteSpace($env:APPDATA)) {
        Add-Candidate $candidates (Join-Path $env:APPDATA "OpenClaw\skills")
    }
    if (-not [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
        Add-Candidate $candidates (Join-Path $env:LOCALAPPDATA "OpenClaw\skills")
    }
    return Find-ExistingPath $candidates.ToArray()
}

function Get-DefaultConfigFile {
    $candidates = [System.Collections.Generic.List[string]]::new()
    Add-Candidate $candidates $env:OPENCLAW_CONFIG
    if (-not [string]::IsNullOrWhiteSpace($env:OPENCLAW_HOME)) {
        Add-Candidate $candidates (Join-Path $env:OPENCLAW_HOME "openclaw.json")
        Add-Candidate $candidates (Join-Path $env:OPENCLAW_HOME "config.json")
    }
    Add-Candidate $candidates (Join-Path $HOME ".openclaw\openclaw.json")
    Add-Candidate $candidates (Join-Path $HOME ".openclaw\config.json")
    if (-not [string]::IsNullOrWhiteSpace($env:APPDATA)) {
        Add-Candidate $candidates (Join-Path $env:APPDATA "OpenClaw\openclaw.json")
        Add-Candidate $candidates (Join-Path $env:APPDATA "OpenClaw\config.json")
    }
    if (-not [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
        Add-Candidate $candidates (Join-Path $env:LOCALAPPDATA "OpenClaw\openclaw.json")
        Add-Candidate $candidates (Join-Path $env:LOCALAPPDATA "OpenClaw\config.json")
    }
    return Find-ExistingPath $candidates.ToArray()
}

function Show-Usage {
    Write-Host "Usage:"
    Write-Host "  pwsh -File install_ant_chrome_openclaw.ps1 -TargetSkillsDir <skills-dir> [-ConfigFile <openclaw.json>] [-SetDefaultProfile]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -TargetSkillsDir     OpenClaw skills directory"
    Write-Host "  -ConfigFile          OpenClaw config path (for example openclaw.json)"
    Write-Host "  -BrowserProfileName  Browser profile name to create or update (default: ant-chrome)"
    Write-Host "  -BaseUrl             Ant Browser LaunchServer base URL"
    Write-Host "  -ApiHeader           API auth header name"
    Write-Host "  -ApiKey              API key written into the skill entry"
    Write-Host "  -Color               Browser profile color (default: #0F766E)"
    Write-Host "  -SetDefaultProfile   Set browser.defaultProfile to the selected profile"
    Write-Host "  -DryRun              Print detected paths without writing files"
}

$skillName = "ant-chrome-openclaw"
$effectiveBaseUrl = Resolve-DefaultValue $BaseUrl (Resolve-DefaultValue $env:ANT_CHROME_BASE_URL "http://127.0.0.1:19876")
$effectiveApiHeader = Resolve-DefaultValue $ApiHeader (Resolve-DefaultValue $env:ANT_CHROME_API_HEADER "X-Ant-Api-Key")
$effectiveApiKey = Resolve-DefaultValue $ApiKey (Resolve-DefaultValue $env:ANT_CHROME_API_KEY "")
$sourceSkillRoot = Get-SourceSkillRoot

if ($Help) {
    Show-Usage
    exit 0
}

$resolvedSkillsDir = Resolve-DefaultValue $TargetSkillsDir (Get-DefaultSkillsDir)
if ([string]::IsNullOrWhiteSpace($resolvedSkillsDir)) {
    throw "TargetSkillsDir is required because no existing OpenClaw skills directory was detected."
}

$resolvedConfigFile = Resolve-DefaultValue $ConfigFile (Get-DefaultConfigFile)
$skillDestination = Join-Path $resolvedSkillsDir $skillName
$backupPath = ""

if ($DryRun) {
    Write-Host "[dry-run] source skill: $sourceSkillRoot"
    Write-Host "[dry-run] target skills dir: $resolvedSkillsDir"
    Write-Host "[dry-run] install destination: $skillDestination"
    if (-not [string]::IsNullOrWhiteSpace($resolvedConfigFile)) {
        Write-Host "[dry-run] config file to update: $resolvedConfigFile"
    } else {
        Write-Host "[dry-run] config file: not set; only files would be installed"
    }
    exit 0
}

[System.IO.Directory]::CreateDirectory($resolvedSkillsDir) | Out-Null

if (Test-Path -LiteralPath $skillDestination) {
    $backupName = "{0}.backup-{1}" -f $skillName, (Get-Date -Format "yyyyMMddHHmmss")
    $backupPath = Join-Path $resolvedSkillsDir $backupName
    Rename-Item -LiteralPath $skillDestination -NewName $backupName
}

Copy-Item -LiteralPath $sourceSkillRoot -Destination $resolvedSkillsDir -Recurse -Force

$configUpdated = $false
if (-not [string]::IsNullOrWhiteSpace($resolvedConfigFile)) {
    $configData = Read-JsonFile -Path $resolvedConfigFile

    $browser = Ensure-MapNode -Parent $configData -Key "browser"
    $browser["enabled"] = $true
    $profiles = Ensure-MapNode -Parent $browser -Key "profiles"
    $profile = Ensure-MapNode -Parent $profiles -Key $BrowserProfileName
    $profile["cdpUrl"] = $effectiveBaseUrl
    if ([string]::IsNullOrWhiteSpace([string]$profile["color"])) {
        $profile["color"] = $Color
    }
    if ($SetDefaultProfile -or [string]::IsNullOrWhiteSpace([string]$browser["defaultProfile"])) {
        $browser["defaultProfile"] = $BrowserProfileName
    }

    $skills = Ensure-MapNode -Parent $configData -Key "skills"
    $entries = Ensure-MapNode -Parent $skills -Key "entries"
    $skillEntry = Ensure-MapNode -Parent $entries -Key $skillName
    $skillEntry["enabled"] = $true
    $envMap = Ensure-MapNode -Parent $skillEntry -Key "env"
    $envMap["ANT_CHROME_BASE_URL"] = $effectiveBaseUrl
    $envMap["ANT_CHROME_API_HEADER"] = $effectiveApiHeader
    if (-not [string]::IsNullOrWhiteSpace($effectiveApiKey)) {
        $skillEntry["apiKey"] = $effectiveApiKey
    }

    Write-JsonFile -Path $resolvedConfigFile -Data $configData
    $configUpdated = $true
}

Write-Host "Installed skill to: $skillDestination"
if (-not [string]::IsNullOrWhiteSpace($backupPath)) {
    Write-Host "Backed up previous skill to: $backupPath"
}
if ($configUpdated) {
    Write-Host "Updated OpenClaw config: $resolvedConfigFile"
} else {
    Write-Host "No config file updated. Merge openclaw.config.sample.json manually or rerun with -ConfigFile."
}
Write-Host "Browser profile name: $BrowserProfileName"
Write-Host "Base URL: $effectiveBaseUrl"
