package utils

import (
	"time"

	"github.com/schollz/progressbar/v3"
)

// NewProgressBar 创建一个新的进度条
func NewProgressBar(total int) *progressbar.ProgressBar {
	return NewProgressBarWithDescription(total, "下载进度")
}

func NewProgressBarWithDescription(total int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(10),
		progressbar.OptionClearOnFinish(),
	)
}

func NewSpinner(description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionClearOnFinish(),
	)
}
