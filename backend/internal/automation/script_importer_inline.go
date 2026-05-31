package automation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func ImportBundleFromBytes(nameHint string, data []byte, sourceLabel string) (ImportedBundle, error) {
	return ImportBundleFromBytesWithOptions(nameHint, data, sourceLabel, ImportOptions{})
}

func ImportBundleFromBytesWithOptions(nameHint string, data []byte, sourceLabel string, options ImportOptions) (ImportedBundle, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return ImportedBundle{}, fmt.Errorf("script content is empty")
	}
	if isZipArchiveData(nameHint, data) {
		return importBundleFromZipBytes(nameHint, data, sourceLabel, options)
	}

	inline, inlineErr := importInlineBundle(filepath.Base(nameHint), data, sourceLabel)
	if inlineErr == nil {
		return inline, nil
	}

	ext := strings.ToLower(filepath.Ext(nameHint))
	if ext == ".json" {
		for _, candidate := range importManifestCandidates {
			if strings.EqualFold(filepath.Base(nameHint), candidate) {
				return ImportedBundle{}, fmt.Errorf("manifest file needs to be imported from a directory or git repository")
			}
		}
		return ImportedBundle{}, inlineErr
	}
	if ext == ".js" || ext == ".cjs" || ext == ".mjs" {
		return importPlainScriptBundle(filepath.Base(nameHint), data, sourceLabel)
	}
	if isTypeScriptSourceFile(nameHint) {
		if !options.AllowTypeScriptBuild {
			return ImportedBundle{}, fmt.Errorf("当前环境未开启 TypeScript 脚本构建支持，暂不支持 .ts / .mts / .cts 入口")
		}
		return importTypeScriptSingleFileBundle(filepath.Base(nameHint), data, sourceLabel)
	}

	for _, candidate := range importManifestCandidates {
		if strings.EqualFold(filepath.Base(nameHint), candidate) {
			return ImportedBundle{}, fmt.Errorf("manifest file needs to be imported from a directory or git repository")
		}
	}

	return importPlainScriptBundle(filepath.Base(nameHint), data, sourceLabel)
}

func importInlineBundle(nameHint string, data []byte, sourceLabel string) (ImportedBundle, error) {
	var envelope scriptImportEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ImportedBundle{}, fmt.Errorf("import content is not valid JSON")
	}

	record, err := buildImportedRecord(envelope, trimExtension(filepath.Base(nameHint)), sourceLabel)
	if err != nil {
		return ImportedBundle{}, err
	}
	record.Notes = appendImportSourceNote(record.Notes, sourceLabel)

	extraFiles, err := decodeScriptTemplateFiles(envelope.Files, record.EntryFile)
	if err != nil {
		return ImportedBundle{}, err
	}

	bundle := ImportedBundle{
		Record: record,
		Files: append([]ImportedBundleFile{
			{
				Path:    record.EntryFile,
				Content: []byte(record.ScriptText),
			},
		}, extraFiles...),
	}
	if err := validateImportedBundle(bundle.Record, bundle.Files); err != nil {
		return ImportedBundle{}, err
	}
	return bundle, nil
}

func importPlainScriptBundle(nameHint string, data []byte, sourceLabel string) (ImportedBundle, error) {
	record, _ := normalizeScriptRecord(ScriptRecord{
		PackageFormat:   defaultScriptPackageFormat,
		ManifestVersion: defaultScriptManifestVersion,
		ID:              uuid.NewString(),
		Name:            trimExtension(filepath.Base(nameHint)),
		Description:     "",
		Type:            "playwright-cdp",
		Status:          "draft",
		EntryFile:       normalizeScriptEntryFile(defaultEntryFileForName(nameHint)),
		ScriptText:      string(data),
		Notes:           appendImportSourceNote("", sourceLabel),
		Source:          inferImportSource(sourceLabel),
		CreatedAt:       time.Now().Format(time.RFC3339),
		UpdatedAt:       time.Now().Format(time.RFC3339),
		SelectorText:    "",
		ParamsText:      "",
	}, ScriptRecord{})

	bundle := ImportedBundle{
		Record: record,
		Files: []ImportedBundleFile{
			{
				Path:    record.EntryFile,
				Content: []byte(record.ScriptText),
			},
		},
	}
	if err := validateImportedBundle(bundle.Record, bundle.Files); err != nil {
		return ImportedBundle{}, err
	}
	return bundle, nil
}
