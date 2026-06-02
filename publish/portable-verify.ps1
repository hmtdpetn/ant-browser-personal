# Ant Browser Personal - prefer_ipv4 修复运行时诊断脚本（测试环境用）
#
# 用法：先在 ant-chrome.exe 里启动那条问题节点，再运行：
#   verify-proxy.bat            （自动读取 socks 端口）
#   verify-proxy.bat 41080      （手动指定 socks 端口）
#
# 本脚本会：
#  1) 找到最新生成的 xray/sing-box config，打印 server / SNI / IPv4 策略字段；
#  2) 【关键】不经代理，直接 TCP 探测节点 server:port 在本网络是否可达；
#  3) 解析节点域名，列出全部 A / AAAA 记录（判断是否 GeoDNS 选了坏 IP）；
#  4) 测本机直连 IPv4 出网是否正常；
#  5) curl 经 socks 端口验证出口 IP；
#  6) tail 内核日志确认拨号目标。

param([int]$Port = 0)

$ErrorActionPreference = "Continue"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$dataDir = Join-Path $root "data"

function Line { Write-Host ("-" * 64) }

Write-Host ""
Write-Host "==== Ant Browser prefer_ipv4 诊断 ===="
Write-Host "包目录: $root"
Write-Host ""

if (-not (Test-Path -LiteralPath $dataDir)) {
    Write-Host "[X] 未找到 data 目录：$dataDir"
    Write-Host "    请先启动 ant-chrome.exe 并运行那条问题节点，再跑本脚本。"
    exit 1
}

# 1) 找最新生成的内核 config（只认这两个文件名，排除日志等其它文件）
$cfgFiles = Get-ChildItem -LiteralPath $dataDir -Recurse -File -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -eq "singbox-config.json" -or $_.Name -eq "xray-config.json" } |
    Sort-Object LastWriteTime -Descending
if (-not $cfgFiles -or @($cfgFiles).Count -eq 0) {
    Write-Host "[X] data 下没有找到 singbox-config.json / xray-config.json。"
    Write-Host "    说明节点还没启动成功，或用的是直连/socks 节点（无需内核）。"
    exit 1
}

$cfgFile = @($cfgFiles)[0]
$isSingBox = $cfgFile.Name -eq "singbox-config.json"
$kernel = if ($isSingBox) { "sing-box" } else { "xray" }
Write-Host "[1] 最新内核配置 ($kernel):"
Write-Host "    $($cfgFile.FullName)"
Write-Host "    生成时间: $($cfgFile.LastWriteTime)"
Line

$cfg = $null
try {
    $cfg = Get-Content -LiteralPath $cfgFile.FullName -Raw | ConvertFrom-Json
} catch {
    Write-Host "[X] 解析配置 JSON 失败：$($_.Exception.Message)"
    exit 1
}

$serverShown = ""   # config 里 server 字段（可能是 IP，也可能是域名）
$sniShown = ""       # SNI / serverName（应为原域名）
$serverPort = 0
$strategyOK = $false
$detectedPort = 0

if ($isSingBox) {
    $proxy = $cfg.outbounds | Where-Object { $_.tag -eq "proxy-out" } | Select-Object -First 1
    if ($null -ne $proxy) {
        $serverShown = [string]$proxy.server
        $serverPort = [int]$proxy.server_port
        if ($proxy.tls) { $sniShown = [string]$proxy.tls.server_name }
        Write-Host "    server          = $serverShown : $serverPort"
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
        if ($vnext) { $serverShown = [string]$vnext[0].address; $serverPort = [int]$vnext[0].port }
        elseif ($servers) { $serverShown = [string]$servers[0].address; $serverPort = [int]$servers[0].port }
        Write-Host "    server address  = $serverShown : $serverPort"
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

# server 是否已是 IPv4 字面量
$isIPv4 = $false
$parsed = [System.Net.IPAddress]::Any
if ([System.Net.IPAddress]::TryParse($serverShown, [ref]$parsed)) {
    $isIPv4 = ($parsed.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetwork)
}
if ($isIPv4) { Write-Host "[OK] config 里 server 已是 IPv4 字面量。" }
else { Write-Host "[i] config 里 server 是 '$serverShown'（域名，未被替换为 IP）。" }
if ($strategyOK) { Write-Host "[OK] IPv4-only 策略字段已写入。" } else { Write-Host "[!] 未检测到 IPv4-only 策略字段。" }
Write-Host ""

# 原始域名：优先用 SNI（被替换时原域名在这里），否则用 server 本身
$origDomain = if ($sniShown -and -not [System.Net.IPAddress]::TryParse($sniShown, [ref]$parsed)) { $sniShown }
              elseif (-not $isIPv4) { $serverShown } else { "" }

Line
Write-Host "[2] 【关键】不经代理，直接 TCP 探测节点端点（在本网络是否可达）"
function Probe-TCP([string]$ip, [int]$p) {
    try {
        $r = Test-NetConnection -ComputerName $ip -Port $p -WarningAction SilentlyContinue
        if ($r.TcpTestSucceeded) { Write-Host "    [OK] $ip : $p 可达 (TCP 连接成功)" }
        else { Write-Host "    [X]  $ip : $p 不可达 (TCP 连接失败/超时)" }
    } catch { Write-Host "    [X]  $ip : $p 探测异常: $($_.Exception.Message)" }
}
if ($isIPv4 -and $serverPort -gt 0) { Probe-TCP $serverShown $serverPort }

Line
Write-Host "[3] 解析节点域名的全部 A / AAAA 记录（判断是否 GeoDNS 选了坏 IP）"
if ($origDomain) {
    Write-Host "    域名: $origDomain"
    try {
        $a = Resolve-DnsName -Name $origDomain -Type A -ErrorAction SilentlyContinue | Where-Object { $_.Type -eq "A" }
        $aaaa = Resolve-DnsName -Name $origDomain -Type AAAA -ErrorAction SilentlyContinue | Where-Object { $_.Type -eq "AAAA" }
        $aips = @($a | ForEach-Object { $_.IPAddress })
        $aaaaips = @($aaaa | ForEach-Object { $_.IPAddress })
        $aShown = if ($aips.Count) { $aips -join ', ' } else { '(无)' }
        $aaaaShown = if ($aaaaips.Count) { $aaaaips -join ', ' } else { '(无)' }
        Write-Host "    A    记录: $aShown"
        Write-Host "    AAAA 记录: $aaaaShown"
        if ($serverPort -gt 0 -and $aips.Count -gt 0) {
            Write-Host "    -> 逐个直连探测每个 A 记录的 $serverPort 端口："
            foreach ($ip in $aips) { Probe-TCP $ip $serverPort }
        }
    } catch { Write-Host "    解析失败: $($_.Exception.Message)" }
} else {
    Write-Host "    (config 里没有可用域名，跳过)"
}

Line
Write-Host "[4] 本机直连 IPv4 出网是否正常（不经代理）"
$myip = & curl.exe -4 -s -m 8 https://api.ipify.org
if ($myip) { Write-Host "    [OK] 直连 IPv4 出口: $myip" }
else { Write-Host "    [X]  直连 IPv4 取不到出口 IP —— 本网络 IPv4 出网可能本身有问题！" }

Line
# 5) curl 验证出口 IP
if ($Port -le 0) { $Port = $detectedPort }
Write-Host "[5] curl 经 socks 127.0.0.1:$Port 验证代理出口 IP"
if ($Port -le 0) {
    Write-Host "    [X] 无法确定 socks 端口，请改用：verify-proxy.bat <端口>"
} else {
    Write-Host "    >> curl -4 --socks5-hostname 127.0.0.1:$Port https://api.ipify.org"
    $ip4 = & curl.exe -4 -s -m 15 --socks5-hostname "127.0.0.1:$Port" https://api.ipify.org
    Write-Host "       结果: $ip4"
    Write-Host "    >> curl    --socks5-hostname 127.0.0.1:$Port https://api.ipify.org"
    $ipd = & curl.exe -s -m 15 --socks5-hostname "127.0.0.1:$Port" https://api.ipify.org
    Write-Host "       结果: $ipd"
}

Line
# 6) tail 内核日志
$logName = if ($isSingBox) { "singbox.log" } else { "xray-error.log" }
$logPath = Join-Path $cfgFile.DirectoryName $logName
Write-Host "[6] 内核日志尾部 ($logName)："
if (Test-Path -LiteralPath $logPath) {
    Get-Content -LiteralPath $logPath -Tail 15 | ForEach-Object { Write-Host "   $_" }
} else {
    Write-Host "   (未找到 $logPath)"
}

Line
Write-Host ""
Write-Host "怎么读这份报告："
Write-Host "  - [2]/[3] 若 server 端口在本网络【直连也不可达】，但你用流量能通"
Write-Host "    => 根因是本网络对该 IPv4 端点不可达（ISP/路由器封锁，或 GeoDNS 给了坏 IP），"
Write-Host "       不是代理配置能修的；可换节点或在 [3] 里找一个能 ping 通的 A 记录。"
Write-Host "  - [4] 若直连 IPv4 出口都取不到 => 本网络 IPv4 出网本身有问题。"
Write-Host "  - [2] 直连可达、但 [5] 经代理超时 => 才是内核/配置问题，把本报告发我。"
Write-Host ""
