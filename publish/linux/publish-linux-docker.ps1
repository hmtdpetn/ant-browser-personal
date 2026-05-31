param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,
    [string]$ArchOutFile,
    [string]$Version,
    [string]$BuilderBaseImage
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$script:DockerContextArgs = @()

function Invoke-Docker {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Args,
        [string[]]$ContextArgs,
        [switch]$Quiet,
        [switch]$AllowFailure
    )

    $effectiveContextArgs = @()
    if ($PSBoundParameters.ContainsKey("ContextArgs")) {
        $effectiveContextArgs = @($ContextArgs)
    }
    else {
        $effectiveContextArgs = @($script:DockerContextArgs)
    }

    if ($Quiet) {
        & docker @effectiveContextArgs @Args 2>$null
    }
    else {
        & docker @effectiveContextArgs @Args
    }
    $code = $LASTEXITCODE
    if ($code -ne 0 -and -not $AllowFailure) {
        $contextSuffix = ""
        if ($effectiveContextArgs.Count -ge 2) {
            $contextSuffix = " [context: $($effectiveContextArgs[1])]"
        }
        throw "docker$contextSuffix $($Args -join ' ') failed with exit code $code"
    }
    return $code
}

function Invoke-DockerWithRetry {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Args,
        [int]$MaxAttempts = 3,
        [int]$DelaySeconds = 5
    )

    $attempt = 0
    while ($attempt -lt $MaxAttempts) {
        $attempt++
        try {
            Invoke-Docker -Args $Args | Out-Null
            return
        }
        catch {
            if ($attempt -ge $MaxAttempts) {
                throw
            }

            Write-Host "  Docker 命令失败，准备重试 ($attempt/$MaxAttempts): $($_.Exception.Message)"
            Start-Sleep -Seconds $DelaySeconds
        }
    }
}

function Get-DockerContextCandidates {
    $candidates = @(
        [pscustomobject]@{
            Label = "current"
            ContextArgs = @()
        }
    )

    $seen = @{}
    $seen["current"] = $true

    $extraNames = @()
    $currentContextName = ((& docker context show 2>$null) | Out-String).Trim()
    if ($LASTEXITCODE -eq 0 -and $currentContextName -ne "") {
        $extraNames += $currentContextName
    }
    $extraNames += @("default", "desktop-linux")

    foreach ($name in $extraNames) {
        $trimmed = ([string]$name).Trim()
        if ($trimmed -eq "" -or $seen.ContainsKey($trimmed)) {
            continue
        }

        try {
            & docker context inspect $trimmed *> $null
        }
        catch {
            continue
        }

        if ($LASTEXITCODE -ne 0) {
            continue
        }

        $seen[$trimmed] = $true
        $candidates += [pscustomobject]@{
            Label = $trimmed
            ContextArgs = @("--context", $trimmed)
        }
    }

    return $candidates
}

function Get-DockerInfo {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Label,
        [AllowEmptyCollection()]
        [string[]]$ContextArgs
    )

    $format = "{{.OSType}}|{{.Architecture}}"
    $output = ""
    $code = 1
    try {
        $output = (& docker @ContextArgs info --format $format 2>$null | Out-String).Trim()
        $code = $LASTEXITCODE
    }
    catch {
        return $null
    }

    if ($code -ne 0 -or $output -eq "") {
        return $null
    }

    $parts = $output -split "\|", 2
    $osType = ""
    $architecture = ""
    if ($parts.Length -ge 1) {
        $osType = $parts[0].Trim().ToLowerInvariant()
    }
    if ($parts.Length -ge 2) {
        $architecture = $parts[1].Trim()
    }

    return [pscustomobject]@{
        Label = $Label
        ContextArgs = @($ContextArgs)
        OSType = $osType
        Architecture = $architecture
    }
}

function Resolve-DockerLinuxContext {
    param(
        [int]$Attempts = 1,
        [int]$DelaySeconds = 0
    )

    $lastReadyContexts = @()

    for ($attempt = 0; $attempt -lt $Attempts; $attempt++) {
        $readyContexts = @()

        foreach ($candidate in Get-DockerContextCandidates) {
            $info = Get-DockerInfo -Label $candidate.Label -ContextArgs $candidate.ContextArgs
            if ($null -eq $info) {
                continue
            }

            $readyContexts += $info
            if ($info.OSType -eq "linux") {
                return [pscustomobject]@{
                    LinuxContext = $info
                    ReadyContexts = $readyContexts
                }
            }
        }

        $lastReadyContexts = $readyContexts
        if ($attempt -lt ($Attempts - 1)) {
            Start-Sleep -Seconds $DelaySeconds
        }
    }

    return [pscustomobject]@{
        LinuxContext = $null
        ReadyContexts = $lastReadyContexts
    }
}

function Get-EnvOrDefault {
    param(
        [string]$Name,
        [string]$DefaultValue
    )

    $value = [Environment]::GetEnvironmentVariable($Name)
    if ($null -eq $value) {
        return $DefaultValue
    }

    $trimmed = $value.Trim()
    if ($trimmed -eq "") {
        return $DefaultValue
    }

    return $trimmed
}

try {
    Write-Host "[Linux] 检查 Docker Desktop 环境..."
    if (-not (Get-Command docker.exe -ErrorAction SilentlyContinue)) {
        throw "未检测到 docker.exe，无法执行 Linux 打包`n  请先安装 Docker Desktop，并启用 Linux 容器引擎"
    }

    $dockerResolution = Resolve-DockerLinuxContext
    if ($null -eq $dockerResolution.LinuxContext) {
        $dockerDesktopExe = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
        if (Test-Path -LiteralPath $dockerDesktopExe) {
            $dockerDesktopRunning = @(Get-Process -Name "Docker Desktop" -ErrorAction SilentlyContinue).Count -gt 0
            if ($dockerDesktopRunning) {
                Write-Host "  Docker Desktop 已在运行，等待 Linux 引擎就绪..."
            }
            else {
                Write-Host "  Docker Linux 引擎未就绪，尝试启动 Docker Desktop..."
                Start-Process -FilePath $dockerDesktopExe | Out-Null
            }
            $dockerResolution = Resolve-DockerLinuxContext -Attempts 24 -DelaySeconds 5
        }
    }

    if ($null -eq $dockerResolution.LinuxContext) {
        $readyContexts = @($dockerResolution.ReadyContexts)
        if ($readyContexts.Count -gt 0) {
            $readySummary = ($readyContexts | ForEach-Object { "$($_.Label) [$($_.OSType)]" }) -join ", "
            throw "Docker 已启动，但当前可用上下文不是 Linux 容器: $readySummary`n  请在 Docker Desktop 中切换到 Linux containers 后重试"
        }

        throw "Docker 引擎未就绪`n  请先启动 Docker Desktop，并确认 Linux 容器引擎可用"
    }

    $script:DockerContextArgs = @($dockerResolution.LinuxContext.ContextArgs)
    $contextLabel = $dockerResolution.LinuxContext.Label
    $dockerArchRaw = ([string]$dockerResolution.LinuxContext.Architecture).Trim()
    $linuxArch = switch ($dockerArchRaw.ToLowerInvariant()) {
        "x86_64" { "amd64" }
        "amd64" { "amd64" }
        "aarch64" { "arm64" }
        "arm64" { "arm64" }
        default { "" }
    }

    if ($linuxArch -eq "") {
        throw "无法识别 Docker Linux 架构`n  当前输出: $dockerArchRaw"
    }

    $repoDocker = ($RepoRoot -replace "\\", "/").Trim()
    if ($repoDocker -eq "") {
        throw "无法转换仓库路径为 Docker 挂载路径"
    }

    Write-Host "✓ Docker Desktop 就绪"
    Write-Host "  Docker 上下文: $contextLabel"
    Write-Host "  仓库挂载: $repoDocker"
    Write-Host "  Linux 架构: $linuxArch"
    Write-Host ""

    $builderImage = "ant-browser-linux-builder:local"
    $dockerfilePath = Join-Path $RepoRoot "publish/linux/linux-builder.Dockerfile"
    $resolvedBuilderBaseImage = $BuilderBaseImage
    if (-not $resolvedBuilderBaseImage -or $resolvedBuilderBaseImage.Trim() -eq "") {
        $resolvedBuilderBaseImage = $env:ANT_BROWSER_LINUX_BUILDER_BASE_IMAGE
    }
    if (-not $resolvedBuilderBaseImage -or $resolvedBuilderBaseImage.Trim() -eq "") {
        $resolvedBuilderBaseImage = "swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.25-bookworm"
    }
    $resolvedBuilderBaseImage = $resolvedBuilderBaseImage.Trim()

    $dockerfileHashRaw = (Get-FileHash -LiteralPath $dockerfilePath -Algorithm SHA256).Hash.ToLowerInvariant()
    $dockerfileHash = [System.BitConverter]::ToString(
        [System.Security.Cryptography.SHA256]::Create().ComputeHash(
            [System.Text.Encoding]::UTF8.GetBytes("$dockerfileHashRaw|$resolvedBuilderBaseImage")
        )
    ).Replace("-", "").ToLowerInvariant()
    $builderHashLabel = "ant.browser.builder.hash"
    $builderHashCurrent = ""
    $builderInspectJson = ""
    $builderInspectExitCode = 1
    try {
        $builderInspectJson = & docker @script:DockerContextArgs image inspect $builderImage 2>$null
        $builderInspectExitCode = $LASTEXITCODE
    }
    catch {
        $builderInspectExitCode = 1
    }

    if ($builderInspectExitCode -eq 0) {
        $builderInspectItems = @($builderInspectJson | Out-String | ConvertFrom-Json)
        if ($builderInspectItems.Count -gt 0) {
            $config = $builderInspectItems[0].Config
            if ($null -ne $config) {
                $labelsProperty = $config.PSObject.Properties["Labels"]
                if ($labelsProperty -and $null -ne $labelsProperty.Value) {
                    $labels = $labelsProperty.Value
                    if ($labels.PSObject.Properties.Name -contains $builderHashLabel) {
                        $builderHashCurrent = [string]$labels.PSObject.Properties[$builderHashLabel].Value
                    }
                }
            }
        }
    }
    $needsBuilderImage = ($builderInspectExitCode -ne 0) -or ($builderHashCurrent -ne $dockerfileHash)

    if ($needsBuilderImage) {
        Write-Host "[Linux] 构建 Linux builder 镜像..."
        Write-Host "  Dockerfile: publish/linux/linux-builder.Dockerfile"
        Write-Host "  基础镜像: $resolvedBuilderBaseImage"
        Write-Host ""
        Invoke-DockerWithRetry -Args @(
            "build",
            "--build-arg", "BUILDER_BASE_IMAGE=$resolvedBuilderBaseImage",
            "--label", "$builderHashLabel=$dockerfileHash",
            "-f", "publish/linux/linux-builder.Dockerfile",
            "-t", $builderImage,
            "publish/linux"
        )
    }
    else {
        Write-Host "[Linux] 复用已有 builder 镜像缓存"
        Write-Host "  镜像: $builderImage"
        Write-Host "  基础镜像: $resolvedBuilderBaseImage"
        Write-Host ""
    }

    $npmCacheVolume = "ant-browser-linux-npm-cache"
    $nodeModulesVolume = "ant-browser-linux-node-modules-$linuxArch"
    $goModCacheVolume = "ant-browser-linux-go-mod-cache"
    $goBuildCacheVolume = "ant-browser-linux-go-build-cache"
    $runtimeCpuLimit = Get-EnvOrDefault -Name "ANT_BROWSER_LINUX_DOCKER_CPUS" -DefaultValue "2"
    $runtimeMemoryLimit = Get-EnvOrDefault -Name "ANT_BROWSER_LINUX_DOCKER_MEMORY" -DefaultValue "1408m"
    $runtimeMemorySwapLimit = Get-EnvOrDefault -Name "ANT_BROWSER_LINUX_DOCKER_MEMORY_SWAP" -DefaultValue "1792m"
    $runtimeShmSize = Get-EnvOrDefault -Name "ANT_BROWSER_LINUX_DOCKER_SHM_SIZE" -DefaultValue "256m"
    $nodeMaxOldSpace = Get-EnvOrDefault -Name "ANT_BROWSER_LINUX_NODE_MAX_OLD_SPACE" -DefaultValue "384"
    $goMaxProcs = Get-EnvOrDefault -Name "ANT_BROWSER_LINUX_GOMAXPROCS" -DefaultValue "2"
    $npmJobs = Get-EnvOrDefault -Name "ANT_BROWSER_LINUX_NPM_JOBS" -DefaultValue "1"
    $versionArgText = ""
    if ($Version -and $Version.Trim() -ne "") {
        $versionArgText = " --version $Version"
    }
    $linuxPublishCommand = 'for f in publish/linux/*.sh tools/runtime/*.sh; do if [ -f "$f" ]; then sed -i ''s/\r$//'' "$f"; fi; done && bash publish/linux/publish-linux.sh --arch {0}{1}' -f $linuxArch, $versionArgText

    Write-Host "[Linux] 通过 Docker Desktop 执行发布脚本..."
    Write-Host "  镜像: $builderImage"
    Write-Host "  挂载: $repoDocker > /workspace"
    Write-Host "  缓存卷: $npmCacheVolume, $nodeModulesVolume, $goModCacheVolume, $goBuildCacheVolume"
    Write-Host "  资源限制: CPUs=$runtimeCpuLimit, Memory=$runtimeMemoryLimit, Swap=$runtimeMemorySwapLimit, /dev/shm=$runtimeShmSize"
    Write-Host "  进程限制: NODE_OPTIONS=--max-old-space-size=$nodeMaxOldSpace, GOMAXPROCS=$goMaxProcs, npm_config_jobs=$npmJobs"
    Write-Host "  命令: 先清理 .sh 换行符，再执行 bash publish/linux/publish-linux.sh --arch $linuxArch$versionArgText"
    Write-Host ""
    Invoke-Docker -Args @(
        "run",
        "--rm",
        "--cpus", $runtimeCpuLimit,
        "--memory", $runtimeMemoryLimit,
        "--memory-swap", $runtimeMemorySwapLimit,
        "--shm-size", $runtimeShmSize,
        "-e", "CI=1",
        "-e", "NODE_OPTIONS=--max-old-space-size=$nodeMaxOldSpace",
        "-e", "GOMAXPROCS=$goMaxProcs",
        "-e", "npm_config_jobs=$npmJobs",
        "-v", "${repoDocker}:/workspace",
        "-v", "${npmCacheVolume}:/root/.npm",
        "-v", "${nodeModulesVolume}:/workspace/frontend/node_modules",
        "-v", "${goModCacheVolume}:/go/pkg/mod",
        "-v", "${goBuildCacheVolume}:/root/.cache/go-build",
        "-w", "/workspace",
        $builderImage,
        "bash",
        "-c",
        $linuxPublishCommand
    ) | Out-Null

    if ($ArchOutFile -and $ArchOutFile.Trim() -ne "") {
        Set-Content -LiteralPath $ArchOutFile -Value $linuxArch -NoNewline -Encoding ascii
    }

    Write-Host "✓ Linux 产物生成成功"
    exit 0
}
catch {
    Write-Host "✗ $($_.Exception.Message)"
    exit 1
}
