package httpclient

import (
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	Client *http.Client
	once   sync.Once
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
