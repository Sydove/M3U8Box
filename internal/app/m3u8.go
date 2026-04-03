package app

import (
	"fmt"
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

func (d *Downloader) Run(links []string) {
	for index, link := range links {
		if err := d.Task(link); err != nil {
			logger.Errorf("第 %d 个链接下载失败: %v", index, err)
			continue
		} else {
			logger.Infof("第 %d 个链接下载完成", index)
		}
		time.Sleep(2 * time.Second)
	}
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
	staticPath := filepath.Join(d.AbsPath, "static")
	if err := utils.EnsureDir(staticPath, true); err != nil {
		return err
	}
	var targetName string
	if d.Name != "" {
		targetName = fmt.Sprintf("%s.mp4", d.Name)
	} else {
		targetName = fmt.Sprintf("%s.mp4", title)
	}
	videoPath := filepath.Join(d.AbsPath, targetName)
	for videoIndex, m3u8URL := range m3u8List {
		m3u8FilePath := filepath.Join(staticPath, fmt.Sprintf("%s_%d.m3u8", taskHash, videoIndex))
		cryptUrl, tsUrlList, err := d.Parser.Parse(m3u8URL, m3u8FilePath)
		if err != nil {
			return err
		}

		// 下载ts文件
		cryptPath, tsFilePathList, err := d.Downloader.DownFile(cryptUrl, tsUrlList, staticPath, taskHash, d.Concurrency)
		if err != nil {
			return err
		}

		// 生成新的m3u8文件
		modifiedM3U8Path, err := d.Merger.Modify(m3u8FilePath, cryptPath, staticPath, tsFilePathList)
		if err != nil {
			return err
		}
		if err := d.Merger.Merge(modifiedM3U8Path, videoPath); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}
