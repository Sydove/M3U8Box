package extractor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/sydove/M3U8Box/internal/logger"
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
	titleRule := regexp.MustCompile(`property="og:title" content="(.*?)"/>`)
	title := titleRule.FindStringSubmatch(strBody)
	titleText := url
	if title == nil || len(title) <= 1 {
		logger.Warnf("%s 标题提取失败", url)
	} else {
		titleText = title[1]
	}

	hrefRule := regexp.MustCompile(`:"(https:.*?.m3u8\?auth_key=.*?)",`)
	hrefs := hrefRule.FindAllStringSubmatch(strBody, -1)
	if hrefs == nil || len(hrefs) == 0 {
		logger.Warnf("%s 链接提取失败", url)
	}
	results := make([]string, 0)
	for _, href := range hrefs {
		if len(href) <= 1 {
			continue
		}
		targetHref := strings.Replace(href[1], "\\", "", -1)
		results = append(results, targetHref)
	}

	if len(results) == 0 {
		return titleText, nil, fmt.Errorf("未提取到 m3u8 链接: %s", url)
	}

	return titleText, results, nil
}

type BrowserhExtractor struct{}

func (e *BrowserhExtractor) Extract(url string) (string, []string, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		return "", nil, err
	}

	m3u8Chan := make(chan string, 20)
	m3u8Map := make(map[string]struct{})

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if res, ok := ev.(*network.EventResponseReceived); ok {
			itemUrl := res.Response.URL
			if strings.Contains(itemUrl, ".m3u8") {
				m3u8Chan <- itemUrl
			}
		}
	})

	done := make(chan struct{})
	go func() {
		for itemUrl := range m3u8Chan {
			if _, exists := m3u8Map[itemUrl]; !exists {
				m3u8Map[itemUrl] = struct{}{}
			}
		}
		close(done)
	}()

	var title string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Title(&title),
		chromedp.Sleep(10*time.Second),
	); err != nil {
		return "", nil, err
	}

	close(m3u8Chan)
	<-done

	var m3u8List []string
	for k := range m3u8Map {
		m3u8List = append(m3u8List, k)
	}

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
