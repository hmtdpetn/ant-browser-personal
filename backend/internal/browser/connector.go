package browser

// BuildLaunchArgs 构建启动参数
func BuildLaunchArgs(args []string, startURLs []string) []string {
	if len(startURLs) == 0 {
		return args
	}
	args = append(args, startURLs...)
	return args
}
