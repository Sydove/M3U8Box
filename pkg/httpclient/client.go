package httpclient

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sydove/M3U8Box/internal/logger"
)

var (
	Client        *http.Client
	once          sync.Once
	RetryAttempts = 3
	RetryDelay    = 2 * time.Second
)

func Init() {
	once.Do(func() {
		Client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		}
	})
}

func DoWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 1; attempt <= RetryAttempts; attempt++ {
		clonedReq := req.Clone(req.Context())
		resp, err := Client.Do(clonedReq)
		if err == nil && !shouldRetryStatus(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("HTTP 响应状态码错误: %d", resp.StatusCode)
			resp.Body.Close()
		}

		if attempt == RetryAttempts {
			break
		}

		logger.Warnf("请求失败，准备重试，第 %d/%d 次: %s, err=%v", attempt, RetryAttempts, req.URL.String(), lastErr)
		time.Sleep(RetryDelay * time.Duration(attempt))
	}

	return nil, lastErr
}

func shouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return statusCode >= 500
	}
}
