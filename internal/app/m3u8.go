package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sydove/M3U8Box/internal/downloader"
	"github.com/sydove/M3U8Box/internal/extractor"
	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/internal/m3u8"
	"github.com/sydove/M3U8Box/internal/merge"
	"github.com/sydove/M3U8Box/internal/utils"
)

type Downloader struct {
	ExtractorChain extractor.ChainExtraction
	Parser         m3u8.Parser
	Downloader     downloader.Downloader
	Merger         merge.Merger
	AbsPath        string
	Concurrency    int
	Name           string
}

func (d *Downloader) Run(links []string) error {
	var runErr error

	for index, link := range links {
		if err := d.Task(link); err != nil {
			logger.Errorf("第 %d 个链接下载失败: %v", index, err)
			runErr = errors.Join(runErr, fmt.Errorf("第 %d 个链接下载失败: %w", index, err))
			continue
		} else {
			logger.Infof("第 %d 个链接下载完成", index)
		}
		time.Sleep(2 * time.Second)
	}
	defer func() {
		if err := d.cleanupStaticDir(); err != nil {
			runErr = errors.Join(runErr, err)
		}
	}()

	return runErr
}

func (d *Downloader) Task(url string) error {
	// 提取m3u8文件链接
	d.ExtractorChain.AddExtractorToChain(&extractor.HLExtractor{})
	d.ExtractorChain.AddExtractorToChain(&extractor.BrowserhExtractor{})
	title, m3u8List, err := d.ExtractorChain.Extract(url)
	if err != nil {
		return err
	}

	// 解析m3u8文件
	taskHash, err := utils.GetTaskHash(title)
	if err != nil {
		return err
	}
	taskStaticPath := filepath.Join(d.AbsPath, "static", taskHash)
	if err := utils.EnsureDir(taskStaticPath, true); err != nil {
		return err
	}
	var targetName string
	for videoIndex, m3u8URL := range m3u8List {
		m3u8FilePath := filepath.Join(taskStaticPath, fmt.Sprintf("%s_%d.m3u8", taskHash, videoIndex))
		cryptUrl, tsUrlList, err := d.Parser.Parse(m3u8URL, m3u8FilePath)
		if err != nil {
			return err
		}

		// 下载ts文件
		cryptPath, tsFilePathList, err := d.Downloader.DownFile(cryptUrl, tsUrlList, taskStaticPath, taskHash, d.Concurrency)
		if err != nil {
			return err
		}

		// 生成新的m3u8文件
		modifiedM3U8Path, err := d.Merger.Modify(m3u8FilePath, cryptPath, taskStaticPath, tsFilePathList)
		if err != nil {
			return err
		}
		if d.Name != "" {
			targetName = fmt.Sprintf("%s_%d.mp4", d.Name, videoIndex)
		} else {
			targetName = fmt.Sprintf("%s_%d.mp4", title, videoIndex)
		}
		videoPath := filepath.Join(d.AbsPath, targetName)
		if err := d.Merger.Merge(modifiedM3U8Path, videoPath); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func (d *Downloader) cleanupStaticDir() error {
	staticPath := filepath.Join(d.AbsPath, "static")
	if err := os.RemoveAll(staticPath); err != nil {
		logger.Warnf("删除 static 目录失败: %s, err=%v", staticPath, err)
		return fmt.Errorf("删除 static 目录失败: %s: %w", staticPath, err)
	}
	logger.Infof("已删除临时目录: %s", staticPath)
	return nil
}
