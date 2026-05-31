package automation

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

func extractImportedModuleSpecifiers(scriptText string) []string {
	specifiers := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)

	appendMatches := func(pattern *regexp.Regexp) {
		for _, match := range pattern.FindAllStringSubmatch(scriptText, -1) {
			if len(match) < 2 {
				continue
			}
			specifier := strings.TrimSpace(match[1])
			if specifier == "" {
				continue
			}
			if _, exists := seen[specifier]; exists {
				continue
			}
			seen[specifier] = struct{}{}
			specifiers = append(specifiers, specifier)
		}
	}

	appendMatches(scriptRequirePattern)
	appendMatches(scriptDynamicImportRegex)
	appendMatches(scriptImportFromPattern)
	appendMatches(scriptExportFromPattern)

	return specifiers
}

func validateImportedSpecifier(importerPath string, specifier string, fileIndex map[string][]byte) (string, error) {
	normalized := strings.TrimSpace(specifier)
	if normalized == "" {
		return "", nil
	}

	if strings.HasPrefix(normalized, "./") || strings.HasPrefix(normalized, "../") {
		return resolveImportedLocalModule(importerPath, normalized, fileIndex)
	}
	if strings.HasPrefix(normalized, "/") || looksLikeWindowsAbsolutePath(normalized) {
		return "", fmt.Errorf("不支持绝对路径依赖 %q", normalized)
	}
	if isAllowedRuntimeModule(normalized) {
		return "", nil
	}
	return "", fmt.Errorf("发现不受支持的外部依赖 %q，只允许相对路径、Node 内置模块、playwright、playwright-core", normalized)
}

func resolveImportedLocalModule(importerPath string, specifier string, fileIndex map[string][]byte) (string, error) {
	candidate := path.Clean(path.Join(path.Dir(importerPath), specifier))
	if candidate == "." || candidate == ".." || strings.HasPrefix(candidate, "../") {
		return "", fmt.Errorf("本地依赖 %q 超出了脚本包范围", specifier)
	}

	if resolved, err := resolveImportedLocalCandidate(candidate, fileIndex); err == nil {
		return resolved, nil
	} else if err != nil {
		return "", fmt.Errorf("本地依赖 %q 无法解析: %w", specifier, err)
	}
	return "", fmt.Errorf("本地依赖 %q 无法解析", specifier)
}

func resolveImportedLocalCandidate(candidate string, fileIndex map[string][]byte) (string, error) {
	if content, exists := fileIndex[candidate]; exists {
		_ = content
		if isSupportedLocalImportFile(candidate) {
			return candidate, nil
		}
		return "", fmt.Errorf("文件 %s 使用了不支持的扩展名 %s", candidate, path.Ext(candidate))
	}

	if path.Ext(candidate) == "" {
		for _, extension := range []string{".js", ".cjs", ".mjs", ".json"} {
			withExtension := candidate + extension
			if _, exists := fileIndex[withExtension]; exists {
				return withExtension, nil
			}
		}
	}

	if hasImportedBundleDir(candidate, fileIndex) {
		for _, extension := range []string{"/index.js", "/index.cjs", "/index.mjs", "/index.json"} {
			indexFile := candidate + extension
			if _, exists := fileIndex[indexFile]; exists {
				return indexFile, nil
			}
		}
	}

	return "", fmt.Errorf("找不到文件")
}

func hasImportedBundleDir(target string, fileIndex map[string][]byte) bool {
	prefix := strings.TrimSuffix(strings.TrimSpace(target), "/") + "/"
	for filePath := range fileIndex {
		if strings.HasPrefix(filePath, prefix) {
			return true
		}
	}
	return false
}

func containsNodeModulesPath(filePath string) bool {
	for _, segment := range strings.Split(filepathToSlash(filePath), "/") {
		if strings.EqualFold(segment, "node_modules") {
			return true
		}
	}
	return false
}

func filepathToSlash(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
}

func isSupportedScriptModuleFile(filePath string) bool {
	_, exists := supportedScriptModuleExtensions[strings.ToLower(path.Ext(strings.TrimSpace(filePath)))]
	return exists
}

func isSupportedLocalImportFile(filePath string) bool {
	_, exists := supportedLocalImportExtensions[strings.ToLower(path.Ext(strings.TrimSpace(filePath)))]
	return exists
}

func isAllowedRuntimeModule(specifier string) bool {
	switch strings.TrimSpace(specifier) {
	case "playwright", "playwright-core":
		return true
	}

	normalized := strings.TrimPrefix(strings.TrimSpace(specifier), "node:")
	if normalized == "" {
		return false
	}
	root := normalized
	if index := strings.Index(root, "/"); index >= 0 {
		root = root[:index]
	}
	_, exists := nodeBuiltinModules[root]
	return exists
}

func looksLikeWindowsAbsolutePath(value string) bool {
	if len(value) < 3 {
		return false
	}
	drive := value[0]
	if !((drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z')) {
		return false
	}
	return value[1] == ':' && (value[2] == '\\' || value[2] == '/')
}
