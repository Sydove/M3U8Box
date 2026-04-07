package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/sydove/M3U8Box/internal/app"
	"github.com/sydove/M3U8Box/internal/downloader"
	"github.com/sydove/M3U8Box/internal/extractor"
	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/internal/m3u8"
	"github.com/sydove/M3U8Box/internal/merge"
	"github.com/sydove/M3U8Box/pkg/httpclient"
)

func main() {
	os.Exit(run())
}

func run() int {
	if err := logger.Init(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		return 1
	}
	defer func() {
		if err := logger.Close(); err != nil {
			log.Printf("关闭日志文件失败: %v", err)
		}
	}()

	httpclient.Init()

	options, err := parseRunOptions()
	if err != nil {
		if errors.Is(err, errShowUsage) {
			printUsage()
			return 1
		}
		logger.Errorf("%v", err)
		printUsage()
		return 1
	}

	merger := &merge.FmgMerger{}
	if options.mode == modePackage {
		packager := app.Packager{
			Merger:      merger,
			AbsPath:     options.dir,
			Name:        options.name,
			SegmentTime: options.hlsTime,
		}
		if err := packager.Run(options.mp4Path); err != nil {
			logger.Errorf("HLS 清单生成失败: %v", err)
			return 1
		}
		return 0
	}

	chain := extractor.ChainExtraction{}
	chain.AddExtractorToChain(&extractor.HLExtractor{})
	chain.AddExtractorToChain(&extractor.BrowserhExtractor{})
	downloadApp := app.Downloader{
		ExtractorChain: chain,
		Parser:         &m3u8.HLParser{},
		Downloader:     &downloader.HLDownloader{},
		Merger:         merger,
		AbsPath:        options.dir,
		Concurrency:    options.concurrency,
		Name:           options.name,
	}
	if err := downloadApp.Run(options.links); err != nil {
		logger.Errorf("任务执行结束，但存在错误: %v", err)
		return 1
	}

	return 0
}
