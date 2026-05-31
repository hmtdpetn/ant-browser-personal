package automation

import (
	"encoding/json"
	"fmt"
	"path"
)

func validateImportedBundle(record ScriptRecord, files []ImportedBundleFile) error {
	fileIndex, err := buildImportedBundleFileIndex(record, files)
	if err != nil {
		return err
	}

	entryFile, err := normalizeBundleFilePath(record.EntryFile)
	if err != nil {
		return fmt.Errorf("脚本入口文件无效: %w", err)
	}
	if !isSupportedScriptModuleFile(entryFile) {
		return fmt.Errorf("脚本入口文件必须是 .js / .cjs / .mjs，当前为 %s", path.Ext(entryFile))
	}

	if err := validateImportedPackageJSONFiles(fileIndex); err != nil {
		return err
	}
	return validateReachableScriptModules(entryFile, fileIndex)
}

func buildImportedBundleFileIndex(record ScriptRecord, files []ImportedBundleFile) (map[string][]byte, error) {
	fileIndex := make(map[string][]byte, len(files)+1)
	for _, file := range files {
		relativePath, err := normalizeBundleFilePath(file.Path)
		if err != nil {
			return nil, err
		}
		if containsNodeModulesPath(relativePath) {
			return nil, fmt.Errorf("脚本包不能包含 node_modules，请改成自包含脚本包")
		}
		fileIndex[relativePath] = file.Content
	}

	entryFile, err := normalizeBundleFilePath(record.EntryFile)
	if err != nil {
		return nil, err
	}
	if _, exists := fileIndex[entryFile]; !exists {
		fileIndex[entryFile] = []byte(record.ScriptText)
	}
	return fileIndex, nil
}

func validateImportedPackageJSONFiles(fileIndex map[string][]byte) error {
	for filePath, content := range fileIndex {
		if path.Base(filePath) != "package.json" {
			continue
		}

		var pkg importedPackageJSON
		if err := json.Unmarshal(content, &pkg); err != nil {
			return fmt.Errorf("%s 不是合法的 package.json: %w", filePath, err)
		}

		switch {
		case len(pkg.Dependencies) > 0:
			return fmt.Errorf("%s 包含 dependencies，当前脚本包不支持外部 npm 依赖", filePath)
		case len(pkg.DevDependencies) > 0:
			return fmt.Errorf("%s 包含 devDependencies，当前脚本包不支持依赖安装流程", filePath)
		case len(pkg.PeerDependencies) > 0:
			return fmt.Errorf("%s 包含 peerDependencies，当前脚本包不支持外部 npm 依赖", filePath)
		case len(pkg.OptionalDependencies) > 0:
			return fmt.Errorf("%s 包含 optionalDependencies，当前脚本包不支持外部 npm 依赖", filePath)
		}
	}
	return nil
}

func validateReachableScriptModules(entryFile string, fileIndex map[string][]byte) error {
	queue := []string{entryFile}
	visited := make(map[string]struct{}, len(fileIndex))

	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		if _, seen := visited[current]; seen {
			continue
		}
		visited[current] = struct{}{}

		content, exists := fileIndex[current]
		if !exists {
			return fmt.Errorf("脚本入口 %s 不存在", current)
		}
		if !isSupportedScriptModuleFile(current) {
			continue
		}

		for _, specifier := range extractImportedModuleSpecifiers(string(content)) {
			resolved, err := validateImportedSpecifier(current, specifier, fileIndex)
			if err != nil {
				return fmt.Errorf("%s: %w", current, err)
			}
			if resolved != "" && isSupportedScriptModuleFile(resolved) {
				queue = append(queue, resolved)
			}
		}
	}

	return nil
}
