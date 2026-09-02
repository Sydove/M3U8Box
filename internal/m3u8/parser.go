package m3u8

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/pkg/httpclient"
)

type Parser interface {
	Parse(m3u8URL string, savePath string) (string, error)
}

type HLParser struct{}

func (p *HLParser) Parse(m3u8URL string, savePath string) (string, error) {
	req, err := http.NewRequest("GET", m3u8URL, nil)
	if err != nil {
		logger.Errorf("创建请求失败: %v", err)
		return "", err
	}
	resp, err := httpclient.DoWithRetry(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP 响应状态码错误: %d", resp.StatusCode)
		logger.Errorf("%v", err)
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取响应体失败: %v", err)
		return "", err
	}

	body = absolutizeRelativeURIs(body, m3u8URL)

	// 保存m3u8文件
	err = os.WriteFile(savePath, body, 0644)
	if err != nil {
		logger.Errorf("保存文件失败: %v", err)
		return "", err
	}
	return savePath, nil
}

func absolutizeRelativeURIs(body []byte, m3u8URL string) []byte {
	baseURL, err := url.Parse(m3u8URL)
	if err != nil {
		logger.Warnf("解析 m3u8 基础地址失败: %v", err)
		return body
	}

	lines := strings.Split(string(body), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			lines[i] = absolutizeQuotedURI(line, baseURL)
			continue
		}

		absoluteURL := resolveURI(trimmed, baseURL)
		if absoluteURL != "" {
			lines[i] = strings.Replace(line, trimmed, absoluteURL, 1)
		}
	}

	return []byte(strings.Join(lines, "\n"))
}

func absolutizeQuotedURI(line string, baseURL *url.URL) string {
	const prefix = `URI="`
	start := strings.Index(line, prefix)
	if start < 0 {
		return line
	}
	valueStart := start + len(prefix)
	valueEnd := strings.Index(line[valueStart:], `"`)
	if valueEnd < 0 {
		return line
	}
	valueEnd += valueStart
	absoluteURL := resolveURI(line[valueStart:valueEnd], baseURL)
	if absoluteURL == "" {
		return line
	}
	return line[:valueStart] + absoluteURL + line[valueEnd:]
}

func resolveURI(rawURI string, baseURL *url.URL) string {
	parsedURI, err := url.Parse(rawURI)
	if err != nil || parsedURI.IsAbs() || strings.HasPrefix(rawURI, "data:") {
		return ""
	}
	return baseURL.ResolveReference(parsedURI).String()
}
