package m3u8

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/sydove/M3U8Box/internal/logger"
	"github.com/sydove/M3U8Box/pkg/httpclient"
)

type Parser interface {
	Parse(m3u8URL string, savePath string) (string, []string, error)
}

type HLParser struct{}

func (p *HLParser) Parse(m3u8URL string, savePath string) (cryptUrl string, tsUrlList []string, err error) {
	req, err := http.NewRequest("GET", m3u8URL, nil)
	if err != nil {
		logger.Errorf("创建请求失败: %v", err)
		return
	}
	resp, err := httpclient.DoWithRetry(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP 响应状态码错误: %d", resp.StatusCode)
		logger.Errorf("%v", err)
		return
	}

	// 提取出所有的.ts视频文件和密钥文件,分别进行下载
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取响应体失败: %v", err)
		return
	}
	strBody := string(body)
	cryptRule := regexp.MustCompile(`URI="(.*?)",IV=`)
	tsRule := regexp.MustCompile(`https?://.*?\.ts\?auth_key=.*?\n`)
	cryptList := cryptRule.FindStringSubmatch(strBody)
	tsList := tsRule.FindAllString(strBody, -1)
	if cryptList == nil || len(cryptList) < 2 {
		logger.Errorf("未解析到视频 crypt 的链接: %s", m3u8URL)
		err = fmt.Errorf("未解析到视频 crypt 的链接: %s", m3u8URL)
		return
	}
	cryptUrl = cryptList[1]
	if tsList == nil || len(tsList) < 1 {
		logger.Errorf("未解析到 ts 文件: %s", m3u8URL)
		err = fmt.Errorf("未解析到 ts 文件: %s", m3u8URL)
		return
	}

	for _, ts := range tsList {
		tsUrlList = append(tsUrlList, ts)
	}

	// 保存m3u8文件
	err = os.WriteFile(savePath, []byte(strBody), 0644)
	if err != nil {
		logger.Errorf("保存文件失败: %v", err)
		return
	}
	return
}
