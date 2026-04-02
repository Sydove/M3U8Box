package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestAllProcess(t *testing.T) {
	data := []string{
		"https://bowl.bdxslpmh.cc/archives/207388.html",
		"https://bowl.bdxslpmh.cc/archives/206193.html",
		"https://bowl.bdxslpmh.cc/archives/205383.html",
		"https://bowl.bdxslpmh.cc/archives/204846.html",
		"https://bowl.bdxslpmh.cc/archives/204653.html",
		"https://bowl.bdxslpmh.cc/archives/204379.html",
		"https://bowl.bdxslpmh.cc/archives/203715.html",
		"https://bowl.bdxslpmh.cc/archives/203362.html",
		"https://bowl.bdxslpmh.cc/archives/203243.html",
		"https://bowl.bdxslpmh.cc/archives/203301.html",
		"https://bowl.bdxslpmh.cc/archives/202555.html",
		"https://bowl.bdxslpmh.cc/archives/202347.html",
		"https://bowl.bdxslpmh.cc/archives/202008.html",
		"https://bowl.bdxslpmh.cc/archives/201964.html",
		"https://bowl.bdxslpmh.cc/archives/199240.html",
		"https://bowl.bdxslpmh.cc/archives/194018.html",
		"https://bowl.bdxslpmh.cc/archives/193990.html",
		"https://bowl.bdxslpmh.cc/archives/179907.html",
		"https://bowl.bdxslpmh.cc/archives/147750.html",
	}

	for _, url := range data {
		var title string
		hrefs := make([]string, 0)
		title, hrefs = getIndexDocument(url)
		if title == "" || len(hrefs) == 0 {
			fmt.Println("请求未解析到m3u8视频链接", url)
		}
		for index, href := range hrefs {
			filename := fmt.Sprintf("%s_%d.mp4", title, index)
			videoSavePath := filepath.Join("/Users/sydove/private/naizijizy", filename)
			syntheticM3U8(href, videoSavePath, "/Users/sydove/private/naizijizy/static", title, 15)
		}
		time.Sleep(10 * time.Second)
	}

}
