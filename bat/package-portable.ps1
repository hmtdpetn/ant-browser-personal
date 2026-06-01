param(
    [string]$Version = "",
    [switch]$SkipBuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

$script:ResolvedVersion = ""

function Write-Section {
    param([string]$Text)
    Write-Host "========================================"
    Write-Host "  $Text"
    Write-Host "========================================"
}

function Get-TrimmedText {
    param([AllowNull()][string]$Value)
    if ($null -eq $Value) { return "" }
    return $Value.Trim()
}

function Resolve-Version {
    param([string]$ExplicitVersion)

    $explicit = Get-TrimmedText $ExplicitVersion
    if ($explicit -ne "") {
        $script:ResolvedVersion = $explicit
        Write-Host "OK version (explicit): $script:ResolvedVersion"
        Write-Host ""
        return
    }

    $wailsConfigPath = Join-Path $repoRoot "wails.json"
    if (-not (Test-Path -LiteralPath $wailsConfigPath -PathType Leaf)) {
        throw "Cannot read version: wails.json not found"
    }
    $wailsConfig = Get-Content -LiteralPath $wailsConfigPath -Raw | ConvertFrom-Json
    $v = Get-TrimmedText ([string]$wailsConfig.info.productVersion)
    if ($v -eq "") {
        throw "Cannot read productVersion from wails.json"
    }
    $script:ResolvedVersion = $v
    Write-Host "OK version (wails.json): $script:ResolvedVersion"
    Write-Host ""
}

function Assert-RuntimeBinaries {
    Write-Host "  Verifying runtime binary SHA256..."
    $manifestPath = Join-Path $repoRoot "publish\runtime-manifest.json"
    if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
        throw "Missing runtime manifest: publish\runtime-manifest.json"
    }

    $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
    $entries = @($manifest.files | Where-Object { $_.targets -contains "windows-amd64" })
    if ($entries.Count -eq 0) {
        throw "No windows-amd64 entries found in runtime manifest"
    }

    $errors = New-Object System.Collections.Generic.List[string]
    foreach ($entry in $entries) {
        $relPath  = Get-TrimmedText ([string]$entry.path)
        $expected = (Get-TrimmedText ([string]$entry.sha256)).ToLowerInvariant()

        if ($relPath -eq "") { continue }
        if ($expected -eq "" -or $expected.Contains("todo_replace")) {
            $errors.Add("${relPath}: sha256 not initialized")
            continue
        }

        $fullPath = Join-Path $repoRoot ($relPath -replace '/', [System.IO.Path]::DirectorySeparatorChar)
        if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
            $errors.Add("${relPath}: file not found")
            continue
        }

        $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $fullPath).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            $errors.Add("${relPath}: SHA256 mismatch (expected $expected, got $actual)")
        }
    }

    if ($errors.Count -gt 0) {
        throw "Runtime binary verification failed:`n  - $($errors -join "`n  - ")"
    }

    Write-Host "OK runtime binaries verified (windows-amd64)"
    Write-Host ""
}

function Invoke-Build {
    $buildScript = Join-Path $PSScriptRoot "build.ps1"
    if (-not (Test-Path -LiteralPath $buildScript -PathType Leaf)) {
        throw "Build script not found: bat\build.ps1"
    }

    Write-Host "  Calling bat\build.ps1..."
    Write-Host ""
    & powershell -NoProfile -ExecutionPolicy Bypass -File $buildScript
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed (exit $LASTEXITCODE)"
    }
    Write-Host ""
    Write-Host "OK build completed"
    Write-Host ""
}

function Assert-BuildOutput {
    $binaryPath = Join-Path $repoRoot "build\bin\ant-chrome.exe"
    if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
        throw "Build output not found: build\bin\ant-chrome.exe`nRun bat\build.bat first, or remove -SkipBuild to build now."
    }
    Write-Host "OK build output ready: build\bin\ant-chrome.exe"
    Write-Host ""
}

function New-PortableStaging {
    Write-Host "  Assembling portable package contents..."

    $stagingRoot = Join-Path $repoRoot "dist\_portable-staging"
    $stagingDir  = Join-Path $stagingRoot "ant-browser-personal"
    $stagingBin  = Join-Path $stagingDir "bin"
    $stagingData = Join-Path $stagingDir "data"

    $exeSrc    = Join-Path $repoRoot "build\bin\ant-chrome.exe"
    $configSrc = Join-Path $repoRoot "publish\config.init.yaml"
    $binDir    = Join-Path $repoRoot "bin"

    if (-not (Test-Path -LiteralPath $exeSrc -PathType Leaf)) {
        throw "Build output missing: build\bin\ant-chrome.exe"
    }
    if (-not (Test-Path -LiteralPath $configSrc -PathType Leaf)) {
        throw "Config template missing: publish\config.init.yaml"
    }

    # Clean and recreate staging on every run to avoid stale files
    if (Test-Path -LiteralPath $stagingRoot) {
        Remove-Item -LiteralPath $stagingRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Path $stagingDir -Force | Out-Null

    # ant-chrome.exe
    Copy-Item -LiteralPath $exeSrc -Destination (Join-Path $stagingDir "ant-chrome.exe") -Force
    Write-Host "  OK ant-chrome.exe"

    # config.yaml  (same source as NSIS publish flow: publish/config.init.yaml)
    Copy-Item -LiteralPath $configSrc -Destination (Join-Path $stagingDir "config.yaml") -Force
    Write-Host "  OK config.yaml  (source: publish\config.init.yaml)"

    # bin/  -- Windows runtime only; linux-* and darwin-* are not included
    New-Item -ItemType Directory -Path $stagingBin -Force | Out-Null
    foreach ($required in @("xray.exe", "sing-box.exe")) {
        $src = Join-Path $binDir $required
        if (-not (Test-Path -LiteralPath $src -PathType Leaf)) {
            throw "Missing runtime binary: bin\$required"
        }
        Copy-Item -LiteralPath $src -Destination (Join-Path $stagingBin $required) -Force
        Write-Host "  OK bin\$required"
    }

    # verify scripts -- run inside the test environment after starting a node
    $verifyBatSrc = Join-Path $repoRoot "publish\portable-verify.bat"
    $verifyPsSrc  = Join-Path $repoRoot "publish\portable-verify.ps1"
    if ((Test-Path -LiteralPath $verifyBatSrc) -and (Test-Path -LiteralPath $verifyPsSrc)) {
        Copy-Item -LiteralPath $verifyBatSrc -Destination (Join-Path $stagingDir "verify-proxy.bat") -Force
        Copy-Item -LiteralPath $verifyPsSrc -Destination (Join-Path $stagingDir "verify-proxy.ps1") -Force
        Write-Host "  OK verify-proxy.bat / verify-proxy.ps1"
    }

    # data/  -- empty placeholder; app initializes on first run
    # Compress-Archive cannot include a truly empty directory, so a .gitkeep marker is used.
    New-Item -ItemType Directory -Path $stagingData -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $stagingData ".gitkeep") -Value "" -Encoding ascii
    Write-Host "  OK data\  (empty dir placeholder, app initializes on first run)"

    Write-Host "OK staging assembled"
    Write-Host ""
    return $stagingRoot
}

function New-PortableZip {
    param(
        [Parameter(Mandatory = $true)]
        [string]$StagingRoot
    )

    $distDir = Join-Path $repoRoot "dist"
    New-Item -ItemType Directory -Path $distDir -Force | Out-Null

    $zipName = "ant-browser-personal-windows-amd64-portable-$script:ResolvedVersion.zip"
    $zipPath = Join-Path $distDir $zipName

    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }

    # Compress from staging parent so the zip root is ant-browser-personal/
    Push-Location $StagingRoot
    try {
        Compress-Archive -Path "ant-browser-personal" -DestinationPath $zipPath
    }
    finally {
        Pop-Location
    }

    if (-not (Test-Path -LiteralPath $zipPath -PathType Leaf)) {
        throw "Zip creation failed: $zipPath"
    }

    $sizeMB = [math]::Round((Get-Item -LiteralPath $zipPath).Length / 1MB, 1)
    Write-Host "OK zip created: dist\$zipName  ($sizeMB MB)"
    Write-Host ""
    return $zipPath
}

function Remove-PortableStaging {
    param([string]$StagingRoot)
    if ($StagingRoot -and (Test-Path -LiteralPath $StagingRoot)) {
        Remove-Item -LiteralPath $StagingRoot -Recurse -Force
        Write-Host "OK staging cleaned up"
        Write-Host ""
    }
}

# ── main ────────────────────────────────────────────────────────────────────

try {
    Write-Section "Ant Browser Personal - Windows Portable Package"
    Write-Host ""
    Write-Host "Repo root: $repoRoot"
    Write-Host ""

    Write-Host "[1/4] Reading version..."
    Resolve-Version -ExplicitVersion $Version

    Write-Host "[2/4] Verifying runtime binaries..."
    Assert-RuntimeBinaries

    if ($SkipBuild) {
        Write-Host "[3/4] Skipping build (-SkipBuild), checking existing output..."
        Assert-BuildOutput
    }
    else {
        Write-Host "[3/4] Building app..."
        Invoke-Build
        Assert-BuildOutput
    }

    Write-Host "[4/4] Assembling and zipping portable package..."
    $stagingRoot = $null
    try {
        $stagingRoot = New-PortableStaging
        $zipPath     = New-PortableZip -StagingRoot $stagingRoot
    }
    finally {
        Remove-PortableStaging -StagingRoot $stagingRoot
    }

    Write-Host ""
    Write-Section "Portable package ready"
    Write-Host ""
    Write-Host "Output: dist\ant-browser-personal-windows-amd64-portable-$script:ResolvedVersion.zip"
    Write-Host ""
    Write-Host "Unzip and run ant-browser-personal\ant-chrome.exe directly."
    Write-Host "Proxy instances require bin\xray.exe and bin\sing-box.exe (included)."
    Write-Host ""
    exit 0
}
catch {
    Write-Host ""
    Write-Section "FAILED"
    Write-Host ""
    Write-Host $_.Exception.Message
    exit 1
}
