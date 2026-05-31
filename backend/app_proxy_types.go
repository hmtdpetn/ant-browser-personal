package backend

// ProxyValidationResult 代理验证结果
type ProxyValidationResult struct {
	Supported bool   `json:"supported"`
	ErrorMsg  string `json:"errorMsg"`
}

// ProxyTestResult 代理测试结果
type ProxyTestResult struct {
	ProxyId   string `json:"proxyId"`
	Ok        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error"`
}

// ProxyIPHealthResult 代理出口 IP 健康信息（透传第三方接口结果）
type ProxyIPHealthResult struct {
	ProxyId        string                 `json:"proxyId"`
	Ok             bool                   `json:"ok"`
	Source         string                 `json:"source"`
	Error          string                 `json:"error"`
	IP             string                 `json:"ip"`
	FraudScore     int64                  `json:"fraudScore"`
	IsResidential  bool                   `json:"isResidential"`
	IsBroadcast    bool                   `json:"isBroadcast"`
	Country        string                 `json:"country"`
	Region         string                 `json:"region"`
	City           string                 `json:"city"`
	AsOrganization string                 `json:"asOrganization"`
	RawData        map[string]interface{} `json:"rawData"`
	UpdatedAt      string                 `json:"updatedAt"`
}
