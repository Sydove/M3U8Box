package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/pkg/httpclient"
)

type Downloader interface {
	CommonDownload(url string, savePath string) error
	DownFile(cryptUrl string, tsList []string, staticPath string, hashStr string, concurrency int) (string, []string, error)
}

type DefaultDownloader struct{}

func (d *DefaultDownloader) CommonDownload(url string, savePath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Errorf("创建请求失败: %v", err)
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	req.Header.Set("Connection", "keep-alive")

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		logger.Errorf("HTTP 请求失败: %v", err)
		logger.Errorf("请求的 URL: %s", url)
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(savePath)
	if err != nil {
		logger.Errorf("创建文件失败: %v", err)
		return err
	}
	defer file.Close()

	buffer := make([]byte, 1024*1024)
	_, err = io.CopyBuffer(file, resp.Body, buffer)
	if err != nil {
		logger.Errorf("读取响应体失败: %v", err)
		return err
	}
	return nil
}

type HLDownloader struct {
	*DefaultDownloader
}

// DownFile 下载单个文件
func (d *HLDownloader) DownFile(cryptUrl string, tsList []string, staticPath string, hashStr string, concurrency int) (string, []string, error) {
	// 下载crypt文件
	cryptSavePath := filepath.Join(staticPath, fmt.Sprintf("crypt.key"))
	if err := d.CommonDownload(cryptUrl, cryptSavePath); err != nil {
		logger.Errorf("下载 crypt 文件失败: %v", err)
		return "", nil, err
	}
	// 下载ts文件
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	tsPath := make([]string, 0, len(tsList))

	// 进度条
	bar := progressbar.NewOptions(len(tsList),
		progressbar.OptionSetDescription("下载进度"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(10), // 限制刷新频率（避免卡顿）
		progressbar.OptionClearOnFinish(),
	)
	bar.RenderBlank()
	os.Stdout.Sync()
	for index, tsUrl := range tsList {
		trimmedUrl := strings.TrimSpace(tsUrl)
		tsSavePath := filepath.Join(staticPath, fmt.Sprintf("%s_%d.ts", hashStr, index))
		tsPath = append(tsPath, tsSavePath)
		wg.Add(1)

		time.Sleep(5 * time.Millisecond)
		go func(url string, path string) {
			defer wg.Done()
			semaphore <- struct{}{}
			if err := d.CommonDownload(url, path); err != nil {
				logger.Errorf("下载 ts 文件失败: %v", err)
			}
			bar.Add(1)
			<-semaphore
		}(trimmedUrl, tsSavePath)
	}
	wg.Wait()
	return cryptSavePath, tsPath, nil
}
