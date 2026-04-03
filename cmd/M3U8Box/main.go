package main

import (
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
	if err := logger.Init(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			log.Printf("关闭日志文件失败: %v", err)
		}
	}()

	httpclient.Init()

	options, err := parseRunOptions()
	if err != nil {
		logger.Errorf("%v", err)
		logger.Infof(usageText())
		os.Exit(1)
	}

	// 运行
	chain := extractor.ChainExtraction{}
	chain.AddExtractorToChain(&extractor.HLExtractor{})
	chain.AddExtractorToChain(&extractor.BrowserhExtractor{})
	downloadApp := app.Downloader{
		ExtractorChain: chain,
		Parser:         &m3u8.HLParser{},
		Downloader:     &downloader.HLDownloader{},
		Merger:         &merge.FmgMerger{},
		AbsPath:        options.dir,
		Concurrency:    options.concurrency,
		Name:           options.name,
	}
	downloadApp.Run(options.links)
}
