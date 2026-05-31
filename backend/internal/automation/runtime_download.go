package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) fetchNodeSHA256(ctx context.Context, url, fileName string) (string, error) {
	body, err := m.fetchText(ctx, url)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		if fields[len(fields)-1] == fileName {
			return strings.TrimSpace(fields[0]), nil
		}
	}
	return "", fmt.Errorf("未找到 Node 归档校验信息: %s", fileName)
}

func (m *Manager) fetchPlaywrightMetadata(ctx context.Context, version string) (playwrightMetadata, error) {
	url := fmt.Sprintf("%s/playwright-core/%s", strings.TrimRight(m.options.NPMRegistryBaseURL, "/"), strings.TrimSpace(version))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return playwrightMetadata{}, err
	}
	resp, err := m.options.HTTPClient.Do(req)
	if err != nil {
		return playwrightMetadata{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return playwrightMetadata{}, fmt.Errorf("metadata request failed: %s", resp.Status)
	}

	var payload struct {
		Dist struct {
			Tarball string `json:"tarball"`
			Shasum  string `json:"shasum"`
		} `json:"dist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return playwrightMetadata{}, err
	}
	if strings.TrimSpace(payload.Dist.Tarball) == "" {
		return playwrightMetadata{}, fmt.Errorf("playwright-core tarball url is empty")
	}
	return playwrightMetadata{
		TarballURL: strings.TrimSpace(payload.Dist.Tarball),
		Shasum:     strings.TrimSpace(payload.Dist.Shasum),
	}, nil
}

func (m *Manager) fetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := m.options.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("request failed: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (m *Manager) downloadFile(ctx context.Context, url, filePath, component string, startProgress, endProgress int, message string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := m.options.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 256*1024)
	m.emitProgress("downloading", startProgress, message, component)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				return err
			}
			written += int64(n)
			if total > 0 {
				progress := startProgress + int(float64(endProgress-startProgress)*(float64(written)/float64(total)))
				m.emitProgress("downloading", progress, message, component)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	m.emitProgress("downloading", endProgress, message, component)
	return nil
}

func (m *Manager) emitProgress(phase string, progress int, message string, component string) {
	if m.emit == nil {
		return
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	m.emit(ProgressEventName, ProgressEvent{
		Phase:     strings.TrimSpace(phase),
		Progress:  progress,
		Message:   strings.TrimSpace(message),
		Component: strings.TrimSpace(component),
	})
}

func (m *Manager) installFailed(err error) error {
	m.mu.Lock()
	m.lastError = err.Error()
	m.mu.Unlock()
	m.emitProgress("error", 0, err.Error(), "")
	return err
}
