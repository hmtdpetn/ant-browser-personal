package automation

import "regexp"

var (
	scriptRequirePattern     = regexp.MustCompile(`\brequire\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	scriptDynamicImportRegex = regexp.MustCompile(`\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	scriptImportFromPattern  = regexp.MustCompile(`(?m)\bimport\s+(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]`)
	scriptExportFromPattern  = regexp.MustCompile(`(?m)\bexport\s+[^'"]*?\s+from\s+['"]([^'"]+)['"]`)
	nodeBuiltinModules       = map[string]struct{}{
		"assert":              {},
		"async_hooks":         {},
		"buffer":              {},
		"child_process":       {},
		"cluster":             {},
		"console":             {},
		"constants":           {},
		"crypto":              {},
		"dgram":               {},
		"diagnostics_channel": {},
		"dns":                 {},
		"domain":              {},
		"events":              {},
		"fs":                  {},
		"http":                {},
		"http2":               {},
		"https":               {},
		"inspector":           {},
		"module":              {},
		"net":                 {},
		"os":                  {},
		"path":                {},
		"perf_hooks":          {},
		"process":             {},
		"punycode":            {},
		"querystring":         {},
		"readline":            {},
		"repl":                {},
		"stream":              {},
		"string_decoder":      {},
		"sys":                 {},
		"timers":              {},
		"tls":                 {},
		"trace_events":        {},
		"tty":                 {},
		"url":                 {},
		"util":                {},
		"v8":                  {},
		"vm":                  {},
		"wasi":                {},
		"worker_threads":      {},
		"zlib":                {},
	}
	supportedScriptModuleExtensions = map[string]struct{}{
		".js":  {},
		".cjs": {},
		".mjs": {},
	}
	supportedLocalImportExtensions = map[string]struct{}{
		".js":   {},
		".cjs":  {},
		".mjs":  {},
		".json": {},
	}
)

type importedPackageJSON struct {
	Dependencies         map[string]any `json:"dependencies"`
	DevDependencies      map[string]any `json:"devDependencies"`
	PeerDependencies     map[string]any `json:"peerDependencies"`
	OptionalDependencies map[string]any `json:"optionalDependencies"`
}
