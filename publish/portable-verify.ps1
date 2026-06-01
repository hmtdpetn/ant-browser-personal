# Ant Browser Personal - prefer_ipv4 修复运行时验证脚本（测试环境用）
#
# 用法：先在 ant-chrome.exe 里启动那条问题节点，使其代理跑起来，然后运行本脚本。
#   verify-proxy.bat            （自动从生成的 config 里读取 socks 端口）
#   verify-proxy.bat 41080      （手动指定 socks 端口）
#
# 脚本会：① 找到最新生成的 xray/sing-box config，打印 server / SNI / IPv4 策略字段；
#         ② 用 curl 走该 socks 端口验证出口 IP；③ tail 内核日志确认拨号目标。

param([int]$Port = 0)

$ErrorActionPreference = "Continue"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$dataDir = Join-Path $root "data"

function Line { Write-Host ("-" * 60) }

Write-Host ""
Write-Host "==== Ant Browser prefer_ipv4 修复验证 ===="
Write-Host "包目录: $root"
Write-Host ""

if (-not (Test-Path -LiteralPath $dataDir)) {
    Write-Host "[X] 未找到 data 目录：$dataDir"
    Write-Host "    请先启动 ant-chrome.exe 并运行那条问题节点，再跑本脚本。"
    exit 1
}

# 1) 找最新生成的内核 config
$cfgFiles = Get-ChildItem -LiteralPath $dataDir -Recurse -Include "singbox-config.json","xray-config.json" -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTime -Descending
if (-not $cfgFiles -or $cfgFiles.Count -eq 0) {
    Write-Host "[X] data 下没有找到 singbox-config.json / xray-config.json。"
    Write-Host "    说明节点还没启动成功，或用的是直连/socks 节点（无需内核）。"
    exit 1
}

$cfgFile = $cfgFiles[0]
$isSingBox = $cfgFile.Name -eq "singbox-config.json"
$kernel = if ($isSingBox) { "sing-box" } else { "xray" }
Write-Host "[1] 最新内核配置 ($kernel):"
Write-Host "    $($cfgFile.FullName)"
Write-Host "    生成时间: $($cfgFile.LastWriteTime)"
Line

$cfg = Get-Content -LiteralPath $cfgFile.FullName -Raw | ConvertFrom-Json

$serverShown = ""
$sniShown = ""
$strategyOK = $false
$detectedPort = 0

if ($isSingBox) {
    $proxy = $cfg.outbounds | Where-Object { $_.tag -eq "proxy-out" } | Select-Object -First 1
    if ($null -ne $proxy) {
        $serverShown = [string]$proxy.server
        if ($proxy.tls) { $sniShown = [string]$proxy.tls.server_name }
        Write-Host "    server          = $serverShown"
        Write-Host "    tls.server_name = $sniShown   (应为原域名)"
        Write-Host "    domain_strategy = $($proxy.domain_strategy)   (应为 ipv4_only)"
    }
    if ($cfg.dns) { Write-Host "    dns.strategy    = $($cfg.dns.strategy)   (应为 ipv4_only)" }
    $strategyOK = ($cfg.dns -and $cfg.dns.strategy -eq "ipv4_only")
    $inbound = $cfg.inbounds | Where-Object { $_.tag -eq "socks-in" } | Select-Object -First 1
    if ($inbound) { $detectedPort = [int]$inbound.listen_port }
}
else {
    $proxy = $cfg.outbounds | Where-Object { $_.tag -eq "proxy-out" } | Select-Object -First 1
    if ($null -ne $proxy -and $proxy.settings) {
        $vnext = $proxy.settings.vnext
        $servers = $proxy.settings.servers
        if ($vnext) { $serverShown = [string]$vnext[0].address }
        elseif ($servers) { $serverShown = [string]$servers[0].address }
        Write-Host "    server address  = $serverShown"
        if ($proxy.streamSettings) {
            $ss = $proxy.streamSettings
            if ($ss.realitySettings) { $sniShown = [string]$ss.realitySettings.serverName }
            elseif ($ss.tlsSettings) { $sniShown = [string]$ss.tlsSettings.serverName }
            Write-Host "    serverName(SNI) = $sniShown   (应为原域名)"
        }
    }
    if ($cfg.dns) { Write-Host "    dns.queryStrategy = $($cfg.dns.queryStrategy)   (应为 UseIPv4)" }
    $strategyOK = ($cfg.dns -and $cfg.dns.queryStrategy -eq "UseIPv4")
    $inbound = $cfg.inbounds | Where-Object { $_.tag -eq "socks-in" } | Select-Object -First 1
    if ($inbound) { $detectedPort = [int]$inbound.port }
}

Line
# server 是否为 IPv4 字面量
$isIPv4 = $false
$tmp = [System.Net.IPAddress]::Any
if ([System.Net.IPAddress]::TryParse($serverShown, [ref]$tmp)) {
    $isIPv4 = ($tmp.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetwork)
}
if ($isIPv4) {
    Write-Host "[OK] server 已是 IPv4 字面量 ($serverShown) —— 内核不可能再拨 IPv6 到节点。"
} else {
    Write-Host "[!] server 仍是 '$serverShown'（非 IPv4 字面量）。"
    Write-Host "    可能：A 记录解析失败已回退原域名（看日志 WARN），或 prefer_ipv4 被关。"
}
if ($strategyOK) { Write-Host "[OK] IPv4-only 解析策略字段已写入。" } else { Write-Host "[!] 未检测到 IPv4-only 策略字段。" }
Write-Host ""

# 2) curl 验证出口 IP
if ($Port -le 0) { $Port = $detectedPort }
if ($Port -le 0) {
    Write-Host "[X] 无法确定 socks 端口，请改用：verify-proxy.bat <端口>"
    exit 1
}
Write-Host "[2] curl 经 socks 127.0.0.1:$Port 验证出口 IP（应返回节点出口 IP，不再超时）"
Line
Write-Host ">> curl -4 --socks5-hostname 127.0.0.1:$Port https://api.ipify.org"
$ip4 = & curl.exe -4 -s -m 15 --socks5-hostname "127.0.0.1:$Port" https://api.ipify.org
Write-Host "   结果: $ip4"
Write-Host ">> curl    --socks5-hostname 127.0.0.1:$Port https://api.ipify.org"
$ipd = & curl.exe -s -m 15 --socks5-hostname "127.0.0.1:$Port" https://api.ipify.org
Write-Host "   结果: $ipd"
Line

# 3) tail 内核日志
$logName = if ($isSingBox) { "singbox.log" } else { "xray-error.log" }
$logPath = Join-Path $cfgFile.DirectoryName $logName
Write-Host "[3] 内核日志尾部 ($logName)："
if (Test-Path -LiteralPath $logPath) {
    Get-Content -LiteralPath $logPath -Tail 15 | ForEach-Object { Write-Host "   $_" }
} else {
    Write-Host "   (未找到 $logPath)"
}
$stderrPath = Join-Path $cfgFile.DirectoryName ($(if ($isSingBox) { "singbox-stderr.log" } else { "xray-stderr.log" }))
if (Test-Path -LiteralPath $stderrPath) {
    Write-Host "   --- stderr 尾部 ---"
    Get-Content -LiteralPath $stderrPath -Tail 8 | ForEach-Object { Write-Host "   $_" }
}
Line

Write-Host ""
Write-Host "判定："
Write-Host "  - 两条 curl 都返回了同一个出口 IP  => 修复在本网络生效。"
Write-Host "  - server 为 IPv4 字面量 + 策略字段已写入 => 配置层面已强制 IPv4。"
Write-Host "  - 若想看对节点 server 的明确拨号行，把 config 里 log 级别改为 debug 再重启节点。"
Write-Host ""
