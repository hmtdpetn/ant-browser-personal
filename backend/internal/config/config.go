package config

import "strings"

const (
	DefaultMaxProfileLimit          = 9999
	StandardCDKeyProfileBonus       = 10
	GithubStarRewardKey             = "GITHUB_STAR_REWARD"
	GithubStarProfileBonus          = 50
	GithubStarProfileTotal          = DefaultMaxProfileLimit + GithubStarProfileBonus
	DefaultLaunchServerPort         = 19876
	DefaultLaunchServerAPIKeyHeader = "X-Ant-Api-Key"
	DefaultAutomationInstallPolicy  = "on_demand"
	DefaultAutomationNodeSource     = "auto"
	DefaultAutomationNodeVersion    = "22.15.1"
	DefaultAutomationPWVersion      = "1.59.0"
)

const (
	AutomationNodeSourceAuto    = "auto"
	AutomationNodeSourceSystem  = "system"
	AutomationNodeSourceBundled = "bundled"
)

// RewardForUsedKey 返回指定兑换记录对应的永久额度奖励。
func RewardForUsedKey(key string) int {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	if normalized == "" {
		return 0
	}
	if normalized == GithubStarRewardKey {
		return GithubStarProfileBonus
	}
	return StandardCDKeyProfileBonus
}

// MinimumProfileLimitForUsedKeys 根据兑换记录计算最低应得实例额度。
func MinimumProfileLimitForUsedKeys(keys []string) int {
	limit := DefaultMaxProfileLimit
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		normalized := strings.ToUpper(strings.TrimSpace(key))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		limit += RewardForUsedKey(normalized)
	}
	return limit
}

// LaunchServerConfig Launch HTTP 服务配置
type LaunchServerConfig struct {
	Port int                    `yaml:"port"`
	Auth LaunchServerAuthConfig `yaml:"auth"`
}

type LaunchServerAuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	APIKey  string `yaml:"api_key"`
	Header  string `yaml:"header"`
}

type AutomationConfig struct {
	Enabled               bool   `yaml:"enabled"`
	InstallPolicy         string `yaml:"install_policy,omitempty"`
	RuntimeVersion        string `yaml:"runtime_version,omitempty"`
	HeadlessDefault       bool   `yaml:"headless_default,omitempty"`
	KeepRuntimeOnDisable  bool   `yaml:"keep_runtime_on_disable,omitempty"`
	AllowTypeScriptBuild  bool   `yaml:"allow_typescript_build,omitempty"`
	NodeSource            string `yaml:"node_source,omitempty"`
	SystemNodePath        string `yaml:"system_node_path,omitempty"`
	NodeVersion           string `yaml:"node_version,omitempty"`
	PlaywrightCoreVersion string `yaml:"playwright_core_version,omitempty"`
}

// Config 应用配置
type Config struct {
	Database     DatabaseConfig     `yaml:"database"`
	App          AppConfig          `yaml:"app"`
	Runtime      RuntimeConfig      `yaml:"runtime"`
	Logging      LoggingConfig      `yaml:"logging"`
	Browser      BrowserConfig      `yaml:"browser"`
	ProxyCheck   ProxyCheckConfig   `yaml:"proxy_check"`
	LaunchServer LaunchServerConfig `yaml:"launch_server"`
	Automation   AutomationConfig   `yaml:"automation"`
}

type ProxyCheckConfig struct {
	BridgeStartTimeoutMs int                `yaml:"bridge_start_timeout_ms" json:"bridgeStartTimeoutMs"`
	SpeedTargetID        string             `yaml:"speed_target_id" json:"speedTargetId"`
	IPHealthTargetID     string             `yaml:"ip_health_target_id" json:"ipHealthTargetId"`
	Targets              []ProxyCheckTarget `yaml:"targets" json:"targets"`
}

type ProxyCheckTarget struct {
	ID             string `yaml:"id" json:"id"`
	Name           string `yaml:"name" json:"name"`
	Type           string `yaml:"type" json:"type"`
	URL            string `yaml:"url" json:"url"`
	Parser         string `yaml:"parser,omitempty" json:"parser,omitempty"`
	TimeoutMs      int    `yaml:"timeout_ms,omitempty" json:"timeoutMs,omitempty"`
	ExpectedStatus []int  `yaml:"expected_status,omitempty" json:"expectedStatus,omitempty"`
}

type DatabaseConfig struct {
	Type   string       `yaml:"type"`
	SQLite SQLiteConfig `yaml:"sqlite"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type AppConfig struct {
	Name            string       `yaml:"name"`
	Window          WindowConfig `yaml:"window"`
	MaxProfileLimit int          `yaml:"max_profile_limit"`
	UsedCDKeys      []string     `yaml:"used_cd_keys"`
	// PreferIPv4 控制是否在生成代理内核配置时强制只用 IPv4 解析节点 server 域名，
	// 避免在下发原生 IPv6 但节点 IPv6 端点不可达的网络环境下出站全部超时。
	// 指针类型用于区分「未配置」（默认开启）与「显式关闭」(prefer_ipv4: false)。
	PreferIPv4 *bool `yaml:"prefer_ipv4,omitempty"`
}

// PreferIPv4Enabled 返回是否启用 IPv4-only 解析策略。
// 未配置（nil）时默认开启；显式设为 false 时恢复原行为。
func (c *Config) PreferIPv4Enabled() bool {
	if c == nil || c.App.PreferIPv4 == nil {
		return true
	}
	return *c.App.PreferIPv4
}

type WindowConfig struct {
	Width     int `yaml:"width"`
	Height    int `yaml:"height"`
	MinWidth  int `yaml:"min_width"`
	MinHeight int `yaml:"min_height"`
}

type RuntimeConfig struct {
	MaxMemoryMB int `yaml:"max_memory_mb"`
	GCPercent   int `yaml:"gc_percent"`
}

type BrowserBookmark struct {
	Name        string `yaml:"name" json:"name"`
	URL         string `yaml:"url" json:"url"`
	OpenOnStart bool   `yaml:"open_on_start,omitempty" json:"openOnStart"`
}

type BrowserConfig struct {
	UserDataRoot           string                 `yaml:"user_data_root"`
	DefaultFingerprintArgs []string               `yaml:"default_fingerprint_args"`
	DefaultLaunchArgs      []string               `yaml:"default_launch_args"`
	DefaultStartURLs       []string               `yaml:"default_start_urls"`
	RestoreLastSession     bool                   `yaml:"restore_last_session"`
	StartReadyTimeoutMs    int                    `yaml:"start_ready_timeout_ms,omitempty"`
	StartStableWindowMs    int                    `yaml:"start_stable_window_ms,omitempty"`
	DefaultBookmarks       []BrowserBookmark      `yaml:"default_bookmarks,omitempty"`
	Cores                  []BrowserCore          `yaml:"cores,omitempty"`
	Proxies                []BrowserProxy         `yaml:"proxies,omitempty"`
	Profiles               []BrowserProfileConfig `yaml:"profiles,omitempty"`
	ChromeBinaryPath       string                 `yaml:"chrome_binary_path,omitempty"`
	ClashBinaryPath        string                 `yaml:"clash_binary_path,omitempty"`
	XrayBinaryPath         string                 `yaml:"xray_binary_path,omitempty"`
	SingBoxBinaryPath      string                 `yaml:"singbox_binary_path,omitempty"`
	CoreRoot               string                 `yaml:"core_root,omitempty"`
	DefaultCoreId          string                 `yaml:"default_core_id,omitempty"`
	DefaultConnectorType   string                 `yaml:"default_connector_type,omitempty"`
	Environments           []BrowserEnvironment   `yaml:"environments,omitempty"`
}

type BrowserCore struct {
	CoreId    string `yaml:"core_id" json:"coreId"`
	CoreName  string `yaml:"core_name" json:"coreName"`
	CorePath  string `yaml:"core_path" json:"corePath"`
	IsDefault bool   `yaml:"is_default" json:"isDefault"`
}

type BrowserProxy struct {
	ProxyId                string `yaml:"proxy_id" json:"proxyId"`
	ProxyName              string `yaml:"proxy_name" json:"proxyName"`
	ProxyConfig            string `yaml:"proxy_config" json:"proxyConfig"`
	DnsServers             string `yaml:"dns_servers,omitempty" json:"dnsServers,omitempty"`
	GroupName              string `yaml:"group_name,omitempty" json:"groupName,omitempty"`
	SortOrder              int    `yaml:"sort_order,omitempty" json:"sortOrder,omitempty"`
	SourceID               string `yaml:"source_id,omitempty" json:"sourceId,omitempty"`
	SourceURL              string `yaml:"source_url,omitempty" json:"sourceUrl,omitempty"`
	SourceNamePrefix       string `yaml:"source_name_prefix,omitempty" json:"sourceNamePrefix,omitempty"`
	SourceAutoRefresh      bool   `yaml:"source_auto_refresh,omitempty" json:"sourceAutoRefresh,omitempty"`
	SourceRefreshIntervalM int    `yaml:"source_refresh_interval_m,omitempty" json:"sourceRefreshIntervalM,omitempty"`
	SourceLastRefreshAt    string `yaml:"source_last_refresh_at,omitempty" json:"sourceLastRefreshAt,omitempty"`
	LastLatencyMs          int64  `yaml:"-" json:"lastLatencyMs"`
	LastTestOk             bool   `yaml:"-" json:"lastTestOk"`
	LastTestedAt           string `yaml:"-" json:"lastTestedAt"`
	LastIPHealthJSON       string `yaml:"-" json:"lastIPHealthJson,omitempty"`
}

type BrowserEnvironment struct {
	CoreId        string `yaml:"core_id" json:"coreId"`
	CoreName      string `yaml:"core_name" json:"coreName"`
	CorePath      string `yaml:"core_path" json:"corePath"`
	ProxyConfig   string `yaml:"proxy_config" json:"proxyConfig"`
	ConnectorType string `yaml:"connector_type" json:"connectorType"`
	IsDefault     bool   `yaml:"is_default" json:"isDefault"`
}

type BrowserProfileConfig struct {
	ProfileId          string   `yaml:"profile_id" json:"profileId"`
	ProfileName        string   `yaml:"profile_name" json:"profileName"`
	UserDataDir        string   `yaml:"user_data_dir" json:"userDataDir"`
	CoreId             string   `yaml:"core_id" json:"coreId"`
	FingerprintArgs    []string `yaml:"fingerprint_args" json:"fingerprintArgs"`
	ProxyId            string   `yaml:"proxy_id" json:"proxyId"`
	ProxyConfig        string   `yaml:"proxy_config" json:"proxyConfig"`
	ProxyBindSourceID  string   `yaml:"proxy_bind_source_id,omitempty" json:"proxyBindSourceId,omitempty"`
	ProxyBindSourceURL string   `yaml:"proxy_bind_source_url,omitempty" json:"proxyBindSourceUrl,omitempty"`
	ProxyBindName      string   `yaml:"proxy_bind_name,omitempty" json:"proxyBindName,omitempty"`
	ProxyBindUpdatedAt string   `yaml:"proxy_bind_updated_at,omitempty" json:"proxyBindUpdatedAt,omitempty"`
	LaunchArgs         []string `yaml:"launch_args" json:"launchArgs"`
	Tags               []string `yaml:"tags" json:"tags"`
	Keywords           []string `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	CreatedAt          string   `yaml:"created_at" json:"createdAt"`
	UpdatedAt          string   `yaml:"updated_at" json:"updatedAt"`
}

type LoggingConfig struct {
	Level           string            `yaml:"level"`
	FileEnabled     bool              `yaml:"file_enabled"`
	FilePath        string            `yaml:"file_path"`
	Format          string            `yaml:"format"`
	BufferSize      int               `yaml:"buffer_size"`
	AsyncQueueSize  int               `yaml:"async_queue_size"`
	FlushIntervalMs int               `yaml:"flush_interval_ms"`
	Rotation        RotationConfig    `yaml:"rotation"`
	Interceptor     InterceptorConfig `yaml:"interceptor"`
}

type RotationConfig struct {
	Enabled      bool   `yaml:"enabled"`
	MaxSizeMB    int    `yaml:"max_size_mb"`
	MaxAge       int    `yaml:"max_age"`
	MaxBackups   int    `yaml:"max_backups"`
	TimeInterval string `yaml:"time_interval"`
}

type InterceptorConfig struct {
	Enabled         bool     `yaml:"enabled"`
	LogParameters   bool     `yaml:"log_parameters"`
	LogResults      bool     `yaml:"log_results"`
	SensitiveFields []string `yaml:"sensitive_fields"`
}
