package browser

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractZipAndStripRoot 解压 ZIP 包，如果其所有文件全被同一个根目录包裹，则剥离这层根目录解压至 dest
// progressCb 为进度回调 (0-100%, statusType_msg)
func extractZipAndStripRoot(zipPath, dest string, progressCb func(int, string)) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if len(r.File) == 0 {
		return fmt.Errorf("空的压缩包")
	}

	// 探测是否存在单一顶层目录
	var rootPrefix string
	hasCommonRoot := true

	for _, f := range r.File {
		cleanName := filepath.ToSlash(f.Name)
		parts := strings.SplitN(cleanName, "/", 2)

		// 检查空名称文件，理论上不该有
		if len(parts) == 0 || parts[0] == "" {
			continue
		}

		if rootPrefix == "" {
			rootPrefix = parts[0] + "/"
		} else if !strings.HasPrefix(cleanName, rootPrefix) && cleanName != strings.TrimSuffix(rootPrefix, "/") {
			hasCommonRoot = false
			break
		}
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	totalFiles := len(r.File)
	for i, f := range r.File {
		// 报告进度 (逢 5% 更新一下)
		percent := int((float64(i) / float64(totalFiles)) * 100)
		if i%50 == 0 {
			progressCb(percent, fmt.Sprintf("正在解压文件 %d / %d...", i+1, totalFiles))
		}

		cleanName := filepath.ToSlash(f.Name)
		if hasCommonRoot {
			if cleanName == rootPrefix || cleanName == strings.TrimSuffix(rootPrefix, "/") {
				// 忽略外包装本层目录条目
				continue
			}
			cleanName = strings.TrimPrefix(cleanName, rootPrefix)
		}

		if cleanName == "" || cleanName == "/" {
			continue
		}

		fpath := filepath.Join(dest, filepath.FromSlash(cleanName))
		// 防止 Zip Slip 漏洞
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("打开解压文件写入失败 %s: %v", fpath, err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("读取压缩包文件失败 %s: %v", f.Name, err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("写入文件流失败 %s: %v", fpath, err)
		}
	}

	progressCb(100, "解压完成！")
	return nil
}
