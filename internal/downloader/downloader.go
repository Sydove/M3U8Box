package downloader

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/internal/utils"
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

	resp, err := httpclient.DoWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempPath := savePath + ".part"
	file, err := os.Create(tempPath)
	if err != nil {
		logger.Errorf("创建文件失败: %v", err)
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()

	buffer := make([]byte, 1024*1024)
	_, err = io.CopyBuffer(file, resp.Body, buffer)
	if err != nil {
		logger.Errorf("读取响应体失败: %v", err)
		return err
	}
	if err = file.Close(); err != nil {
		logger.Errorf("关闭文件失败: %v", err)
		return err
	}
	closed = true
	if err = os.Rename(tempPath, savePath); err != nil {
		logger.Errorf("重命名临时文件失败: %v", err)
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
	logger.Infof("开始下载视频!")
	// 下载ts文件
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	tsPath := make([]string, 0, len(tsList))
	errChan := make(chan error, len(tsList))

	// 进度条
	bar := utils.NewProgressBar(len(tsList))
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
				errChan <- fmt.Errorf("下载 ts 文件失败: %s: %w", url, err)
			}
			bar.Add(1)
			<-semaphore
		}(trimmedUrl, tsSavePath)
	}
	wg.Wait()
	close(errChan)

	var downloadErr error
	for err := range errChan {
		downloadErr = errors.Join(downloadErr, err)
	}
	if downloadErr != nil {
		return "", nil, downloadErr
	}

	return cryptSavePath, tsPath, nil
}
