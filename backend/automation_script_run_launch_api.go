package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ant-chrome/backend/internal/automation"
)

const dualInstanceRuntimeDefaultTimeoutMs = 45000

type dualInstanceRuntimeParams struct {
	Browsers             []dualInstanceRuntimeBrowserInput `json:"browsers"`
	TimeoutMs            int                               `json:"timeoutMs"`
	SkipDefaultStartURLs *bool                             `json:"skipDefaultStartUrls"`
	PrimaryCode          string                            `json:"primaryCode"`
	SecondaryCode        string                            `json:"secondaryCode"`
}

type dualInstanceRuntimeBrowserInput struct {
	Code                 string   `json:"code"`
	LaunchCode           string   `json:"launchCode"`
	SkipDefaultStartURLs *bool    `json:"skipDefaultStartUrls"`
	StartURLs            []string `json:"startUrls"`
	LaunchArgs           []string `json:"launchArgs"`
}

type dualInstanceRuntimeBrowser struct {
	Code                 string
	SkipDefaultStartURLs bool
	StartURLs            []string
	LaunchArgs           []string
}

func (a *App) runLaunchAPIScript(ctx context.Context, script automation.ScriptRecord, input automation.ScriptRunRequest) (string, string, string) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", "脚本执行失败", automationRunContextErrorMessage(err)
	}
	paramsText := resolveAutomationRunJSONText(input.ParamsText, script.ParamsText, input.UseScriptParams)
	if script.ID == automation.DualInstanceRuntimeScriptID {
		return a.runDualInstanceRuntimeLaunchAPIScript(ctx, paramsText)
	}

	selector, targetSummary, err := a.resolveAutomationEffectiveSelector(script, input, true)
	if err != nil {
		return "", "脚本执行失败", err.Error()
	}
	params, err := parseAutomationJSONObject(paramsText, false)
	if err != nil {
		return "", "脚本执行失败", err.Error()
	}

	body := make(map[string]any, len(params)+1)
	body["selector"] = selector
	for key, value := range params {
		body[key] = value
	}

	status, payload, reqErr := a.automationDemoRequestWithContext(ctx, http.MethodPost, automationDemoLaunchPath, body)
	if reqErr != nil {
		if err := ctx.Err(); err != nil {
			return "", "Launch API 请求失败", automationRunContextErrorMessage(err)
		}
		return "", "Launch API 请求失败", reqErr.Error()
	}

	responseText := marshalAutomationResultText(payload)
	ok := status >= http.StatusOK && status < http.StatusMultipleChoices
	if rawOK, exists := payload["ok"]; exists {
		if payloadOK, valid := rawOK.(bool); valid {
			ok = ok && payloadOK
		}
	}

	summary := appendAutomationRunSummary(fmt.Sprintf("Launch API 响应 HTTP %d", status), targetSummary)
	if ok {
		return responseText, summary, ""
	}

	errorText := mapStringValue(payload, "error")
	if errorText == "" {
		errorText = fmt.Sprintf("launch api returned http %d", status)
	}
	return responseText, summary, errorText
}

func (a *App) runDualInstanceRuntimeLaunchAPIScript(ctx context.Context, paramsText string) (string, string, string) {
	if ctx == nil {
		ctx = context.Background()
	}
	browsers, timeoutMs, err := parseDualInstanceRuntimeParams(paramsText)
	if err != nil {
		return "", "脚本执行失败", err.Error()
	}

	sessions := make([]map[string]interface{}, 0, len(browsers))
	browserCodes := make([]string, 0, len(browsers))

	for _, browser := range browsers {
		if err := ctx.Err(); err != nil {
			return buildDualInstanceRuntimeFailureResult(
				sessions,
				browserCodes,
				"双实例流程超时",
				automationRunContextErrorMessage(err),
			)
		}
		sessionStatus, sessionPayload, reqErr := a.automationDemoRequestWithContext(
			ctx,
			http.MethodPost,
			automationDemoRuntimeSessionPath,
			map[string]any{
				"selector": map[string]any{
					"code":      browser.Code,
					"matchMode": "unique",
				},
				"skipDefaultStartUrls": browser.SkipDefaultStartURLs,
				"startUrls":            browser.StartURLs,
				"launchArgs":           browser.LaunchArgs,
				"timeoutMs":            timeoutMs,
			},
		)
		sessionPayload = ensureAutomationPayload(sessionPayload, browser.Code)
		sessions = append(sessions, sessionPayload)
		if reqErr != nil {
			if err := ctx.Err(); err != nil {
				return buildDualInstanceRuntimeFailureResult(
					sessions,
					browserCodes,
					"双实例流程超时",
					automationRunContextErrorMessage(err),
				)
			}
			return buildDualInstanceRuntimeFailureResult(
				sessions,
				browserCodes,
				fmt.Sprintf("准备 %s Runtime 失败", browser.Code),
				reqErr.Error(),
			)
		}
		if !isAutomationDemoRequestOK(sessionStatus, sessionPayload) {
			errText := mapStringValue(sessionPayload, "error")
			if errText == "" {
				errText = fmt.Sprintf("runtime session api returned http %d", sessionStatus)
			}
			return buildDualInstanceRuntimeFailureResult(
				sessions,
				browserCodes,
				fmt.Sprintf("准备 %s Runtime 失败", browser.Code),
				errText,
			)
		}

		browserCodes = append(browserCodes, browser.Code)
	}

	summary := fmt.Sprintf(
		"%d 个浏览器已就绪：%s",
		len(browserCodes),
		strings.Join(browserCodes, " / "),
	)
	result := map[string]any{
		"ok":           true,
		"summary":      summary,
		"browserCodes": browserCodes,
		"sessions":     sessions,
	}
	return marshalAutomationResultText(result), summary, ""
}

func parseDualInstanceRuntimeParams(paramsText string) ([]dualInstanceRuntimeBrowser, int, error) {
	normalizedText := strings.TrimSpace(paramsText)
	if normalizedText == "" {
		normalizedText = "{}"
	}

	var payload dualInstanceRuntimeParams
	if err := json.Unmarshal([]byte(normalizedText), &payload); err != nil {
		return nil, 0, fmt.Errorf("invalid json object: %w", err)
	}

	defaultSkipDefaultStartURLs := true
	if payload.SkipDefaultStartURLs != nil {
		defaultSkipDefaultStartURLs = *payload.SkipDefaultStartURLs
	}

	browsers := make([]dualInstanceRuntimeBrowser, 0, len(payload.Browsers))
	for index, item := range payload.Browsers {
		code := normalizeDualInstanceRuntimeCode(item.Code)
		if code == "" {
			code = normalizeDualInstanceRuntimeCode(item.LaunchCode)
		}
		if code == "" && index < 2 {
			code = dualInstanceRuntimeDefaultCode(index)
		}
		if code == "" {
			continue
		}

		skipDefaultStartURLs := defaultSkipDefaultStartURLs
		if item.SkipDefaultStartURLs != nil {
			skipDefaultStartURLs = *item.SkipDefaultStartURLs
		}
		startURLs := normalizeDualInstanceRuntimeStrings(item.StartURLs)
		if len(startURLs) == 0 {
			startURLs = dualInstanceRuntimeDefaultStartURLs(index)
		}

		browsers = append(browsers, dualInstanceRuntimeBrowser{
			Code:                 code,
			SkipDefaultStartURLs: skipDefaultStartURLs,
			StartURLs:            startURLs,
			LaunchArgs:           normalizeDualInstanceRuntimeStrings(item.LaunchArgs),
		})
	}

	if len(browsers) == 0 {
		for index, code := range []string{
			normalizeDualInstanceRuntimeCode(payload.PrimaryCode),
			normalizeDualInstanceRuntimeCode(payload.SecondaryCode),
		} {
			if code == "" {
				continue
			}
			browsers = append(browsers, dualInstanceRuntimeBrowser{
				Code:                 code,
				SkipDefaultStartURLs: defaultSkipDefaultStartURLs,
				StartURLs:            dualInstanceRuntimeDefaultStartURLs(index),
			})
		}
	}

	if len(browsers) == 0 {
		browsers = append(browsers,
			dualInstanceRuntimeBrowser{
				Code:                 dualInstanceRuntimeDefaultCode(0),
				SkipDefaultStartURLs: defaultSkipDefaultStartURLs,
				StartURLs:            dualInstanceRuntimeDefaultStartURLs(0),
			},
			dualInstanceRuntimeBrowser{
				Code:                 dualInstanceRuntimeDefaultCode(1),
				SkipDefaultStartURLs: defaultSkipDefaultStartURLs,
				StartURLs:            dualInstanceRuntimeDefaultStartURLs(1),
			},
		)
	}

	timeoutMs := dualInstanceRuntimeDefaultTimeoutMs
	if payload.TimeoutMs > 0 {
		timeoutMs = payload.TimeoutMs
		if timeoutMs < 1000 {
			timeoutMs = 1000
		}
	}

	return browsers, timeoutMs, nil
}

func normalizeDualInstanceRuntimeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeDualInstanceRuntimeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

func dualInstanceRuntimeDefaultCode(index int) string {
	switch index {
	case 0:
		return "BUYER_001"
	case 1:
		return "BUYER_002"
	default:
		return ""
	}
}

func dualInstanceRuntimeDefaultStartURLs(index int) []string {
	switch index {
	case 0:
		return []string{"https://finance.sina.com.cn/"}
	case 1:
		return []string{"https://map.baidu.com/"}
	default:
		return nil
	}
}

func ensureAutomationPayload(payload map[string]interface{}, requestedCode string) map[string]interface{} {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if strings.TrimSpace(requestedCode) != "" {
		payload["requestedCode"] = strings.ToUpper(strings.TrimSpace(requestedCode))
	}
	return payload
}

func isAutomationDemoRequestOK(status int, payload map[string]interface{}) bool {
	ok := status >= http.StatusOK && status < http.StatusMultipleChoices
	if rawOK, exists := payload["ok"]; exists {
		if payloadOK, valid := rawOK.(bool); valid {
			ok = ok && payloadOK
		}
	}
	return ok
}

func buildDualInstanceRuntimeFailureResult(
	sessions []map[string]interface{},
	browserCodes []string,
	step string,
	errorText string,
) (string, string, string) {
	result := map[string]any{
		"ok":           false,
		"summary":      "双实例流程执行失败",
		"error":        errorText,
		"step":         step,
		"browserCodes": browserCodes,
		"sessions":     sessions,
	}
	return marshalAutomationResultText(result), "双实例流程执行失败", errorText
}
