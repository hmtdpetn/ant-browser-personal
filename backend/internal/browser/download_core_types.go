package browser

import "context"

// DownloadProgress 进度信息载体
type DownloadProgress struct {
	Phase    string `json:"phase"`    // "downloading" 或 "extracting" 或 "done" 或 "error"
	Progress int    `json:"progress"` // 进度百分比 0-100
	Message  string `json:"message"`  // 附加详情
}

type coreDownloadWriter struct {
	writeFunc func(p []byte) (n int, err error)
	ctx       context.Context
}

func (cw *coreDownloadWriter) Write(p []byte) (int, error) {
	select {
	case <-cw.ctx.Done():
		return 0, cw.ctx.Err()
	default:
	}
	return cw.writeFunc(p)
}
