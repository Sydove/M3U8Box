package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestAllProcess(t *testing.T) {
	baseUrl := "https://cgw666.com"
	data := []string{
		"/archives/43462/",
		"/archives/43250/",
	}

	for _, suffix := range data {
		if suffix == "" {
			continue
		}
		url := baseUrl + suffix
		title, hrefs := extraM3u8(url)
		if title == "" || len(hrefs) == 0 {
			t.Errorf("extraM3u8 函数执行失败")
		}
		for index, href := range hrefs {
			filename := fmt.Sprintf("%s_%d.mp4", title, index)
			videoSavePath := filepath.Join("/Users/sydove/changtui", filename)
			syntheticM3U8(href, videoSavePath, "/Users/sydove/changtui/static", title, 15)
		}
		time.Sleep(10 * time.Second)
	}

}
