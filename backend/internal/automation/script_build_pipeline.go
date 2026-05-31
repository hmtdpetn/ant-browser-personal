package automation

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

type ImportOptions struct {
	AllowTypeScriptBuild bool
}

func importTypeScriptSingleFileBundle(nameHint string, data []byte, sourceLabel string) (ImportedBundle, error) {
	fileName := filepath.Base(strings.TrimSpace(nameHint))
	if !isTypeScriptSourceFile(fileName) {
		return ImportedBundle{}, fmt.Errorf("TypeScript source file is required")
	}

	tempDir, err := os.MkdirTemp("", "ant-automation-ts-file-*")
	if err != nil {
		return ImportedBundle{}, fmt.Errorf("create typescript temp dir failed: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(sourcePath, data, 0o644); err != nil {
		return ImportedBundle{}, fmt.Errorf("write typescript source failed: %w", err)
	}

	return importTypeScriptDirectoryBundle(tempDir, sourcePath, map[string]any{
		"name":      trimExtension(fileName),
		"type":      "playwright-cdp",
		"entryFile": fileName,
	}, sourceLabel)
}

func importTypeScriptDirectoryBundle(packageRoot string, entryPath string, descriptor map[string]any, sourceLabel string) (ImportedBundle, error) {
	sourceFiles, err := collectImportedBundleFiles(packageRoot)
	if err != nil {
		return ImportedBundle{}, err
	}
	if err := validateTypeScriptSourceFiles(sourceFiles); err != nil {
		return ImportedBundle{}, err
	}

	entryPath = filepath.Clean(entryPath)
	entryRelPath, err := filepath.Rel(packageRoot, entryPath)
	if err != nil {
		return ImportedBundle{}, fmt.Errorf("resolve typescript entry file failed: %w", err)
	}
	entryRelPath = filepath.ToSlash(entryRelPath)

	compiledEntryFile, compiledEntryContent, err := buildTypeScriptEntry(packageRoot, entryRelPath)
	if err != nil {
		return ImportedBundle{}, err
	}

	record, err := buildImportedRecord(scriptImportEnvelope{
		Format:          mapStringValueAny(descriptor, "format"),
		PackageFormat:   mapStringValueAny(descriptor, "packageFormat"),
		ManifestVersion: mapIntValueAny(descriptor, "manifestVersion"),
		Name:            mapStringValueAny(descriptor, "name"),
		Description:     mapStringValueAny(descriptor, "description"),
		Type:            mapStringValueAny(descriptor, "type"),
		Status:          mapStringValueAny(descriptor, "status"),
		EntryFile:       compiledEntryFile,
		Tags:            mapStringSliceValue(descriptor, "tags"),
		Selector:        descriptor["selector"],
		SelectorText:    descriptor["selectorText"],
		Params:          descriptor["params"],
		ParamsText:      descriptor["paramsText"],
		ScriptText:      string(compiledEntryContent),
		Notes:           appendTypeScriptBuildNote(mapStringValueAny(descriptor, "notes")),
		Source:          mapObjectValue(descriptor, "source"),
	}, filepath.Base(packageRoot), sourceLabel)
	if err != nil {
		return ImportedBundle{}, err
	}

	files := make([]ImportedBundleFile, 0, len(sourceFiles)+1)
	files = append(files, ImportedBundleFile{
		Path:    compiledEntryFile,
		Content: compiledEntryContent,
	})
	for _, file := range sourceFiles {
		relativePath, err := normalizeBundleFilePath(file.Path)
		if err != nil {
			return ImportedBundle{}, err
		}
		if relativePath == compiledEntryFile || isTypeScriptSourceFile(relativePath) {
			continue
		}
		files = append(files, ImportedBundleFile{
			Path:    relativePath,
			Content: file.Content,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	bundle := ImportedBundle{
		Record: record,
		Files:  files,
	}
	if err := validateImportedBundle(bundle.Record, bundle.Files); err != nil {
		return ImportedBundle{}, err
	}
	return bundle, nil
}

func validateTypeScriptSourceFiles(files []ImportedBundleFile) error {
	fileIndex := make(map[string][]byte, len(files))
	for _, file := range files {
		relativePath, err := normalizeBundleFilePath(file.Path)
		if err != nil {
			return err
		}
		fileIndex[relativePath] = file.Content
	}
	return validateImportedPackageJSONFiles(fileIndex)
}

func buildTypeScriptEntry(packageRoot string, entryRelPath string) (string, []byte, error) {
	compiledEntryFile := compiledTypeScriptEntryFile(entryRelPath)
	entryAbsPath := filepath.Join(packageRoot, filepath.FromSlash(entryRelPath))
	outfile := filepath.Join(packageRoot, filepath.FromSlash(compiledEntryFile))

	result := api.Build(api.BuildOptions{
		AbsWorkingDir:  packageRoot,
		EntryPoints:    []string{entryAbsPath},
		Outfile:        outfile,
		Bundle:         true,
		Write:          false,
		Platform:       api.PlatformNode,
		Format:         api.FormatCommonJS,
		Target:         api.ES2020,
		Sourcemap:      api.SourceMapNone,
		SourcesContent: api.SourcesContentExclude,
		LegalComments:  api.LegalCommentsNone,
		LogLevel:       api.LogLevelSilent,
		Plugins: []api.Plugin{
			buildTypeScriptImportGuardPlugin(),
		},
	})
	if len(result.Errors) > 0 {
		return "", nil, fmt.Errorf("TypeScript 构建失败: %s", formatTypeScriptBuildMessages(result.Errors))
	}

	for _, output := range result.OutputFiles {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(output.Path)), ".map") {
			continue
		}
		return compiledEntryFile, output.Contents, nil
	}

	return "", nil, fmt.Errorf("TypeScript 构建失败: 未生成输出文件")
}

func buildTypeScriptImportGuardPlugin() api.Plugin {
	return api.Plugin{
		Name: "automation-typescript-import-guard",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^[^./].*`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if args.Kind == api.ResolveEntryPoint {
					return api.OnResolveResult{}, nil
				}
				specifier := strings.TrimSpace(args.Path)
				if isAllowedRuntimeModule(specifier) {
					return api.OnResolveResult{
						Path:     specifier,
						External: true,
					}, nil
				}
				return api.OnResolveResult{}, fmt.Errorf("发现不受支持的外部依赖 %q，只允许相对路径、Node 内置模块、playwright、playwright-core", specifier)
			})
			build.OnResolve(api.OnResolveOptions{Filter: `^(?:/|[A-Za-z]:[\\/])`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				if args.Kind == api.ResolveEntryPoint {
					return api.OnResolveResult{}, nil
				}
				return api.OnResolveResult{}, fmt.Errorf("不支持绝对路径依赖 %q", strings.TrimSpace(args.Path))
			})
		},
	}
}

func compiledTypeScriptEntryFile(entryRelPath string) string {
	normalized := path.Clean(filepath.ToSlash(strings.TrimSpace(entryRelPath)))
	dir := path.Dir(normalized)
	baseName := strings.TrimSuffix(path.Base(normalized), path.Ext(normalized))
	if baseName == "" || baseName == "." {
		baseName = "index"
	}

	compiledName := baseName + ".cjs"
	if dir == "." || dir == "/" || dir == "" {
		return compiledName
	}
	return path.Join(dir, compiledName)
}

func isTypeScriptSourceFile(filePath string) bool {
	switch strings.ToLower(path.Ext(strings.TrimSpace(filepath.ToSlash(filePath)))) {
	case ".ts", ".cts", ".mts":
		return true
	default:
		return false
	}
}

func appendTypeScriptBuildNote(notes string) string {
	line := "构建: TypeScript -> CommonJS"
	trimmed := strings.TrimSpace(notes)
	if trimmed == "" {
		return line
	}
	if strings.Contains(trimmed, line) {
		return trimmed
	}
	return trimmed + "\n" + line
}

func formatTypeScriptBuildMessages(messages []api.Message) string {
	if len(messages) == 0 {
		return "unknown build error"
	}

	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Text)
		if message.Location != nil && strings.TrimSpace(message.Location.File) != "" {
			text = fmt.Sprintf("%s:%d:%d %s", filepath.ToSlash(message.Location.File), message.Location.Line, message.Location.Column, text)
		}
		if text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "unknown build error"
	}
	return strings.Join(parts, "; ")
}
