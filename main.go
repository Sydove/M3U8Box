package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		// 配置代理服务器
		Proxy: func(_ *http.Request) (*url.URL, error) {
			// 替换为你的代理服务器地址和端口
			return url.Parse("http://127.0.0.1:1082")
		},
	},
	Timeout: 1 * time.Minute,
}

func main() {
	var (
		url         string
		dir         string
		name        string
		concurrency int
	)

	// 定义命令行参数
	flag.StringVar(&url, "url", "", "访问的URL地址")
	flag.StringVar(&dir, "dir", ".", "保存的文件目录，默认当前目录")
	flag.StringVar(&name, "name", "", "保存的文件名，默认使用视频标题")
	flag.IntVar(&concurrency, "concurrency", 10, "并发下载数，默认10")
	flag.Parse()

	// 检查URL参数
	if url == "" {
		fmt.Println("错误: 必须指定URL地址")
		fmt.Println("使用方法: M3U8Box --url=<URL> [--dir=<目录>]")
		os.Exit(1)
	}

	// 确保目录存在
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	// 解析url请求中的m3u8列表
	fmt.Printf("正解析: %s\n", url)
	title, m3u8List := extraM3u8(url)
	if m3u8List == nil {
		fmt.Println("未解析到m3u8视频链接")
		os.Exit(1)
	}

	fmt.Printf("正在下载: %s\n", url)
	fmt.Printf("保存目录: %s\n", dir)
	filename := ""
	if name != "" {
		filename = name
	} else {
		filename = title
	}
	for index, file := range m3u8List {
		fullPath := filepath.Join(dir, fmt.Sprintf("%s_%03d.mp4", filename, index))
		syntheticM3U8(file, fullPath, dir, title, concurrency)
	}

	fmt.Println("下载完成!")
}

func extraM3u8(url string) (string, []string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return "", nil
	}

	// 添加 headers，模拟浏览器请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")

	// 发送请求
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("HTTP 请求失败: %v\n", err)
		fmt.Printf("请求的 URL: %s\n", url)
		return "", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("HTTP 响应状态码错误: %d\n", resp.StatusCode)
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return "", nil
	}

	strBody := string(body)
	// 提取文件名称,提取所有m3u8的链接
	titleRule := regexp.MustCompile(`property="og:title" content="(.*?)"/>`)
	title := titleRule.FindStringSubmatch(strBody)
	if title == nil || len(title) <= 1 {
		fmt.Printf("%s标题提取失败\n", url)
	}

	hrefRule := regexp.MustCompile(`:"(https:.*?.m3u8\?auth_key=.*?)",`)
	hrefs := hrefRule.FindAllStringSubmatch(strBody, -1)
	if hrefs == nil || len(hrefs) == 0 {
		fmt.Printf("%s链接提取失败\n", url)
	}
	results := make([]string, 0)
	for _, href := range hrefs {
		if len(href) <= 1 {
			continue
		}
		targetHref := strings.Replace(href[1], "\\", "", -1)
		results = append(results, targetHref)
	}

	return title[1], results
}

func syntheticM3U8(url string, videoSavePath string, savePath string, title string, concurrency int) string {
	timeStr := time.Now().Format("150405")
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s", title, timeStr)))
	hashStr := hex.EncodeToString(hash[:])
	fullPath := filepath.Join(savePath, hashStr)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		os.MkdirAll(fullPath, 0755)
	}
	fmt.Println("开始下载视频文件")
	cryptUrl, tsList, m3u8FilePath, err := extractDownFile(url, fullPath, hashStr)
	if err != nil {
		return ""
	}
	cryptPath, tsPath := getAllFile(cryptUrl, tsList, fullPath, hashStr, concurrency)
	if cryptPath == "" || tsPath == nil || len(tsPath) == 0 {
		return ""
	}
	fmt.Println("开始修改m3u8文件")
	// 修改原来的.m3u8文件,替换为本地路径的所有本地的.ts文件和密钥文件
	modifiedM3U8, err := modifyM3U8(m3u8FilePath, cryptPath, tsPath)
	if err != nil {
		fmt.Printf("修改 m3u8 文件失败: %v\n", err)
		return ""
	}

	// 保存修改后的 m3u8 文件
	modifiedM3U8Path := filepath.Join(fullPath, "modified.m3u8")
	err = os.WriteFile(modifiedM3U8Path, []byte(modifiedM3U8), 0644)
	if err != nil {
		fmt.Printf("保存修改后的 m3u8 文件失败: %v\n", err)
		return ""
	}
	fmt.Println("开始整合视频文件")
	// 构建 ffmpeg 命令
	args := []string{
		"-allowed_extensions", "ALL", // 允许所有扩展
		"-protocol_whitelist", "file,http,https,tcp,tls,crypto", // 白名单
		"-i", modifiedM3U8Path, // 输入 m3u8
		"-c", "copy", // 直接拷贝码流
		"-bsf:a", "aac_adtstoasc", // 音频处理
		videoSavePath, // 输出 mp4
	}

	cmd := exec.Command("ffmpeg", args...)

	// 将命令的标准输出和标准错误连接到终端
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		fmt.Printf("执行 ffmpeg 命令失败: %v\n", err)
		return ""
	}

	fmt.Printf("视频整合完成: %s\n", savePath)
	return savePath
}

func modifyM3U8(m3u8Path, cryptPath string, tsPaths []string) (string, error) {
	// 读取原始 m3u8 文件
	content, err := os.ReadFile(m3u8Path)
	if err != nil {
		return "", err
	}

	m3u8Content := string(content)

	// 替换 crypt.key 路径
	//if cryptPath != "" {
	//	// 使用正则表达式替换 URI 中的密钥路径
	//	cryptRule := regexp.MustCompile(`URI="[^"]+"`)
	//	m3u8Content = cryptRule.ReplaceAllString(m3u8Content, fmt.Sprintf(`URI="%s"`, cryptPath))
	//}

	// 替换 TS 文件路径
	if len(tsPaths) > 0 {
		// 提取原始 TS 文件路径模式
		tsRule := regexp.MustCompile(`https?://[^\n]+\.ts[^\n]*`)
		tsMatches := tsRule.FindAllString(m3u8Content, -1)

		// 替换每个 TS 文件路径
		for i, tsUrl := range tsMatches {
			if i < len(tsPaths) {
				m3u8Content = strings.Replace(m3u8Content, tsUrl, tsPaths[i], 1)
			}
		}
	}

	return m3u8Content, nil
}

func extractDownFile(url string, fullPath string, hastStr string) (string, []string, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return "", nil, "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("HTTP 请求失败: %v\n", err)
		fmt.Printf("请求的 URL: %s\n", url)
		return "", nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("HTTP 响应状态码错误: %d\n", resp.StatusCode)
		return "", nil, "", fmt.Errorf("HTTP 响应状态码错误: %d", resp.StatusCode)
	}

	// 提取出所有的.ts视频文件和密钥文件,分别进行下载
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return "", nil, "", err
	}
	strBody := string(body)
	cryptRule := regexp.MustCompile(`URI="(.*?)",IV=`)
	tsRule := regexp.MustCompile(`https?://.*?\.ts\?auth_key=.*?\n`)
	cryptList := cryptRule.FindStringSubmatch(strBody)
	tsList := tsRule.FindAllString(strBody, -1)
	if cryptList == nil || len(cryptList) < 1 {
		fmt.Printf("未解析到视频crypt的链接:%s\n", url)
		return "", nil, "", nil
	}
	cryptUrl := cryptList[1]
	if tsList == nil || len(tsList) < 1 {
		fmt.Printf("未解析到ts文件:%s\n", url)
	}

	tsUrlList := make([]string, 0, len(tsList))
	for _, ts := range tsList {
		tsUrlList = append(tsUrlList, ts)
	}

	// 保存m3u8文件
	m3u8FilePath := filepath.Join(fullPath, fmt.Sprintf("%s.m3u8", hastStr))
	err = os.WriteFile(m3u8FilePath, []byte(strBody), 0644)
	if err != nil {
		fmt.Printf("保存文件失败: %v\n", err)
		return "", nil, "", err
	}
	return cryptUrl, tsUrlList, m3u8FilePath, nil
}

func getAllFile(cryptUrl string, tsList []string, fullPath string, hashStr string, concurrency int) (string, []string) {
	// 下载crypt文件
	cryptSavePath := filepath.Join(fullPath, fmt.Sprintf("crypt.key"))
	if err := downFile(cryptUrl, cryptSavePath); err != nil {
		fmt.Printf("下载crypt文件失败: %v\n", err)
		return "", nil
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
		tsSavePath := filepath.Join(fullPath, fmt.Sprintf("%s_%d.ts", hashStr, index))
		tsPath = append(tsPath, tsSavePath)
		wg.Add(1)

		time.Sleep(5 * time.Millisecond)
		go func(url string, path string) {
			defer wg.Done()
			semaphore <- struct{}{}
			if err := downFile(url, path); err != nil {
				fmt.Printf("下载ts文件失败: %v\n", err)
			}
			bar.Add(1)
			<-semaphore
		}(trimmedUrl, tsSavePath)
	}
	wg.Wait()
	return cryptSavePath, tsPath
}

func downFile(url string, savePath string) error {

	// 创建请求并添加 headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	req.Header.Set("Connection", "keep-alive")

	// 发送请求
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("HTTP 请求失败: %v\n", err)
		fmt.Printf("请求的 URL: %s\n", url)
		return err
	}
	defer resp.Body.Close()

	// 直接写入文件，避免字符串转换
	file, err := os.Create(savePath)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return err
	}
	defer file.Close()

	// 分块读取并写入
	buffer := make([]byte, 1024*1024) // 1MB 缓冲区
	_, err = io.CopyBuffer(file, resp.Body, buffer)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return err
	}
	return nil
}
