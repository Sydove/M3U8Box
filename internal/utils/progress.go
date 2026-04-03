package utils

import (
	"github.com/schollz/progressbar/v3"
)

// NewProgressBar 创建一个新的进度条
func NewProgressBar(total int) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription("下载进度"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(10),
		progressbar.OptionClearOnFinish(),
	)
}
