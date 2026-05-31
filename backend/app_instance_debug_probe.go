package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const browserDebugProbeTimeout = 250 * time.Millisecond

func probeBrowserDebugPort(debugPort int, requestTimeout time.Duration) error {
	if debugPort <= 0 {
		return fmt.Errorf("invalid debug port %d", debugPort)
	}

	client := &http.Client{Timeout: requestTimeout}
	versionErr := probeBrowserJSONVersion(client, debugPort)
	if versionErr == nil {
		return nil
	}

	listErr := probeBrowserJSONList(client, debugPort)
	if listErr == nil {
		return nil
	}

	return fmt.Errorf("%v; %v", versionErr, listErr)
}

func probeBrowserJSONVersion(client *http.Client, debugPort int) error {
	var payload struct {
		Browser              string `json:"Browser"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := fetchBrowserDebugJSON(client, debugPort, "/json/version", &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Browser) == "" && strings.TrimSpace(payload.WebSocketDebuggerURL) == "" {
		return fmt.Errorf("/json/version missing Browser and webSocketDebuggerUrl")
	}
	return nil
}

func probeBrowserJSONList(client *http.Client, debugPort int) error {
	var payload []map[string]interface{}
	return fetchBrowserDebugJSON(client, debugPort, "/json/list", &payload)
}

func fetchBrowserDebugJSON(client *http.Client, debugPort int, path string, dest interface{}) error {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", debugPort, path)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 256*1024))
	if err := decoder.Decode(dest); err != nil {
		return fmt.Errorf("%s returned invalid JSON: %w", path, err)
	}
	return nil
}

func canConnectDebugPort(debugPort int, dialTimeout time.Duration) bool {
	if debugPort <= 0 {
		return false
	}

	address := fmt.Sprintf("127.0.0.1:%d", debugPort)
	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
