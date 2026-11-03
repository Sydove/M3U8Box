package extractor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/internal/utils"
	"github.com/sydove/M3U8Box/pkg/httpclient"
)

type Extractor interface {
	Extract(pageURL string) (string, []string, error)
}

type HLExtractor struct{}

func (e *HLExtractor) Extract(url string) (string, []string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Errorf("创建请求失败: %v", err)
		return "", nil, err
	}

	// 添加 headers，模拟浏览器请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("path", "/archives/196104/")
	req.Header.Set("Cookie", "__web_sdk_sid_v1=%7B%22sid%22%3A%221d43939290975cd07fad75c8be4484c8%22%2C%22lastBehaviorAt%22%3A1793676701034%7D")

	// 发送请求
	resp, err := httpclient.DoWithRetry(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("HTTP 响应状态码错误: %d", resp.StatusCode)
		return "", nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取响应体失败: %v", err)
		return "", nil, err
	}

	strBody := string(body)
	// 提取文件名称,提取所有m3u8的链接
	var title []string
	baseTitleRule := regexp.MustCompile(`<title>(.*?)</title>`)
	title = baseTitleRule.FindStringSubmatch(strBody)
	if title == nil {
		titleRule := regexp.MustCompile(`property="og:title" content="(.*?)"/>`)
		title = titleRule.FindStringSubmatch(strBody)
	}

	titleText := url
	if title == nil || len(title) <= 1 {
		logger.Warnf("%s 标题提取失败", url)
	} else {
		titleText = title[1]
	}

	hrefRule := regexp.MustCompile(`https?:\\?/\\?/[^"'\s<>]+?\.m3u8(?:\?[^"'\s<>]*)?`)
	hrefs := hrefRule.FindAllStringSubmatch(strBody, -1)
	if hrefs == nil || len(hrefs) == 0 {
		logger.Warnf("%s 链接提取失败", url)
	}
	uniqueSet := utils.NewSet[string]()
	for _, href := range hrefs {
		targetHref := strings.ReplaceAll(href[0], "\\/", "/")
		targetHref = strings.ReplaceAll(targetHref, "\\", "")
		uniqueSet.Add(targetHref)
	}
	results := uniqueSet.GetSlice()
	if len(results) == 0 {
		return titleText, nil, fmt.Errorf("未提取到 m3u8 链接: %s", url)
	}

	return titleText, results, nil
}

type BrowserhExtractor struct{}

func (e *BrowserhExtractor) Extract(url string) (string, []string, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := chromedp.Run(ctx); err != nil {
		return "", nil, err
	}

	m3u8Map := make(map[string]struct{})
	var m3u8Mu sync.Mutex
	var foundOnce sync.Once
	foundM3U8 := make(chan struct{})

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if res, ok := ev.(*network.EventResponseReceived); ok {
			itemUrl := res.Response.URL
			if strings.Contains(itemUrl, ".m3u8") {
				m3u8Mu.Lock()
				m3u8Map[itemUrl] = struct{}{}
				m3u8Mu.Unlock()
				foundOnce.Do(func() { close(foundM3U8) })
			}
		}
	})

	var title string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Title(&title),
	); err != nil {
		return "", nil, err
	}

	m3u8Mu.Lock()
	foundCount := len(m3u8Map)
	m3u8Mu.Unlock()
	if foundCount == 0 {
		select {
		case <-foundM3U8:
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return "", nil, ctx.Err()
		}
	}

	uniqueSet := utils.NewSet[string]()
	m3u8Mu.Lock()
	for k := range m3u8Map {
		uniqueSet.Add(k)
	}
	m3u8Mu.Unlock()
	m3u8List := uniqueSet.GetSlice()

	if len(m3u8List) == 0 {
		return title, nil, fmt.Errorf("浏览器模式未捕获到 m3u8 请求: %s", url)
	}

	return title, m3u8List, nil
}

type ChainExtraction struct {
	extractors []Extractor
}

func (ce *ChainExtraction) AddExtractorToChain(extractors ...Extractor) {
	ce.extractors = append(ce.extractors, extractors...)
}

func (ce *ChainExtraction) Extract(url string) (string, []string, error) {
	var lastErr error

	for _, extractor := range ce.extractors {
		title, m3u8List, err := extractor.Extract(url)
		if err == nil && len(m3u8List) > 0 {
			return title, m3u8List, nil
		}
		lastErr = err
	}
	return "", nil, fmt.Errorf("所有提取器都失败: %v", lastErr)
}
