package backend

import (
	"encoding/json"
	"fmt"
	"strings"
)

func resolveAutomationRunJSONText(value string, fallback string, useFallback bool) string {
	if useFallback {
		return strings.TrimSpace(fallback)
	}
	return strings.TrimSpace(value)
}

func parseAutomationJSONObject(text string, required bool) (map[string]any, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		if required {
			return nil, fmt.Errorf("selector is required")
		}
		return map[string]any{}, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid json object: %w", err)
	}
	return decoded, nil
}

func marshalAutomationResultText(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}
