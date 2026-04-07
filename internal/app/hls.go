package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/internal/merge"
	"github.com/sydove/M3U8Box/internal/utils"
)

type Packager struct {
	Merger      merge.Merger
	AbsPath     string
	Name        string
	SegmentTime int
}

func (p *Packager) Run(mp4Path string) error {
	targetName := p.Name
	if targetName == "" {
		targetName = strings.TrimSuffix(filepath.Base(mp4Path), filepath.Ext(mp4Path))
	}

	outputDir := filepath.Join(p.AbsPath, targetName)
	if err := utils.EnsureDir(outputDir, true); err != nil {
		return fmt.Errorf("创建 HLS 输出目录失败: %w", err)
	}

	playlistPath := filepath.Join(outputDir, "index.m3u8")
	if _, err := os.Stat(playlistPath); err == nil {
		return fmt.Errorf("目标清单已存在: %s", playlistPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查目标清单失败: %w", err)
	}

	logger.Infof("开始将 MP4 生成 HLS 清单: %s", mp4Path)
	if err := p.Merger.Package(mp4Path, playlistPath, p.SegmentTime); err != nil {
		return err
	}

	logger.Infof("HLS 清单生成完成: %s", playlistPath)
	return nil
}
