package logger

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ValidateRotatedFileName 验证文件名是否符合分片命名格式
// 格式: {basename}.{timestamp}[.{sequence}].log
func ValidateRotatedFileName(fileName string) bool {
	// 匹配模式: name.YYYY-MM-DD.log 或 name.YYYY-MM-DD.N.log 或 name.YYYY-MM-DD-HH.log
	patterns := []string{
		`^.+\.\d{4}-\d{2}-\d{2}\.log$`,            // app.2024-01-15.log
		`^.+\.\d{4}-\d{2}-\d{2}\.\d+\.log$`,       // app.2024-01-15.1.log
		`^.+\.\d{4}-\d{2}-\d{2}-\d{2}\.log$`,      // app.2024-01-15-14.log (hourly)
		`^.+\.\d{4}-\d{2}-\d{2}-\d{2}\.\d+\.log$`, // app.2024-01-15-14.1.log
	}

	for _, p := range patterns {
		matched, _ := regexp.MatchString(p, fileName)
		if matched {
			return true
		}
	}
	return false
}

// ParseRotatedFileName 解析分片文件名
// 返回基础名、时间戳、序号
func ParseRotatedFileName(fileName string) (baseName string, timestamp time.Time, sequence int, err error) {
	ext := filepath.Ext(fileName)
	nameWithoutExt := strings.TrimSuffix(fileName, ext)

	// 尝试匹配带序号的格式: app.2024-01-15.1
	seqPattern := regexp.MustCompile(`^(.+)\.(\d{4}-\d{2}-\d{2}(?:-\d{2})?)\.(\d+)$`)
	if matches := seqPattern.FindStringSubmatch(nameWithoutExt); matches != nil {
		baseName = matches[1]
		timestamp, err = parseTimestamp(matches[2])
		if err != nil {
			return "", time.Time{}, 0, err
		}
		fmt.Sscanf(matches[3], "%d", &sequence)
		return baseName, timestamp, sequence, nil
	}

	// 尝试匹配不带序号的格式: app.2024-01-15
	noSeqPattern := regexp.MustCompile(`^(.+)\.(\d{4}-\d{2}-\d{2}(?:-\d{2})?)$`)
	if matches := noSeqPattern.FindStringSubmatch(nameWithoutExt); matches != nil {
		baseName = matches[1]
		timestamp, err = parseTimestamp(matches[2])
		if err != nil {
			return "", time.Time{}, 0, err
		}
		return baseName, timestamp, 0, nil
	}

	return "", time.Time{}, 0, fmt.Errorf("invalid rotated file name format: %s", fileName)
}

// parseTimestamp 解析时间戳字符串
func parseTimestamp(s string) (time.Time, error) {
	// 尝试小时格式
	if t, err := time.Parse("2006-01-02-15", s); err == nil {
		return t, nil
	}
	// 尝试日期格式
	return time.Parse("2006-01-02", s)
}
