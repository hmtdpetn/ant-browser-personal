package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func doConcurrentDownload(ctx context.Context, client *http.Client, targetUrl string, tempFile *os.File, sendEvent func(string, int, string)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return fmt.Errorf("HTTP状态码异常: %d", resp.StatusCode)
	}

	var totalSize int64 = resp.ContentLength
	supportRange := resp.StatusCode == http.StatusPartialContent

	if supportRange {
		cr := resp.Header.Get("Content-Range")
		if cr != "" {
			parts := strings.Split(cr, "/")
			if len(parts) == 2 {
				fmt.Sscanf(parts[1], "%d", &totalSize)
			}
		}
	}
	resp.Body.Close()

	if totalSize <= 0 || !supportRange {
		sendEvent("downloading", 0, "服务器不支持多线程，回退至单流下载...")
		return doSingleThreadDownload(ctx, client, targetUrl, tempFile, totalSize, sendEvent)
	}

	sendEvent("downloading", 0, fmt.Sprintf("支持多线程分片下载，总大小 %.2f MB", float64(totalSize)/1024/1024))

	if err := tempFile.Truncate(totalSize); err != nil {
		return err
	}

	numWorkers := 8
	chunkSize := totalSize / int64(numWorkers)

	var wg sync.WaitGroup
	var downloaded int64
	var mu sync.Mutex
	var lastTick time.Time
	var downloadErr error

	for i := 0; i < numWorkers; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == numWorkers-1 {
			end = totalSize - 1
		}

		wg.Add(1)
		go func(part int, start, end int64) {
			defer wg.Done()

			for retry := 0; retry < 3; retry++ {
				if ctx.Err() != nil {
					return
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetUrl, nil)
				if err != nil {
					mu.Lock()
					if downloadErr == nil {
						downloadErr = err
					}
					mu.Unlock()
					return
				}
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
				pResp, err := client.Do(req)
				if err != nil {
					time.Sleep(2 * time.Second)
					continue
				}

				buf := make([]byte, 256*1024)
				var written int64

				for {
					if ctx.Err() != nil {
						pResp.Body.Close()
						return
					}
					n, rErr := pResp.Body.Read(buf)
					if n > 0 {
						tempFile.WriteAt(buf[:n], start+written)
						written += int64(n)

						mu.Lock()
						downloaded += int64(n)
						if time.Since(lastTick) > time.Second {
							percent := int((float64(downloaded) / float64(totalSize)) * 100)
							sendEvent("downloading", percent, fmt.Sprintf("并行下载中... %.2f MB / %.2f MB", float64(downloaded)/1024/1024, float64(totalSize)/1024/1024))
							lastTick = time.Now()
						}
						mu.Unlock()
					}
					if rErr == io.EOF {
						break
					}
					if rErr != nil {
						mu.Lock()
						if downloadErr == nil {
							downloadErr = rErr
						}
						mu.Unlock()
						pResp.Body.Close()
						return
					}
				}
				pResp.Body.Close()
				return
			}
		}(i, start, end)
	}

	wg.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return downloadErr
}

func doSingleThreadDownload(ctx context.Context, client *http.Client, targetUrl string, tempFile *os.File, totalSize int64, sendEvent func(string, int, string)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetUrl, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态码异常: %d", resp.StatusCode)
	}

	var downloaded int64
	var lastTick time.Time

	pw := &coreDownloadWriter{
		writeFunc: func(p []byte) (n int, err error) {
			n, err = tempFile.Write(p)
			if n > 0 {
				downloaded += int64(n)
				if totalSize > 0 && time.Since(lastTick) > time.Second {
					percent := int((float64(downloaded) / float64(totalSize)) * 100)
					sendEvent("downloading", percent, fmt.Sprintf("单流下载中... %.2f MB / %.2f MB", float64(downloaded)/1024/1024, float64(totalSize)/1024/1024))
					lastTick = time.Now()
				}
			}
			return n, err
		},
		ctx: ctx,
	}

	buf := make([]byte, 1024*1024)
	_, err = io.CopyBuffer(pw, resp.Body, buf)
	return err
}
