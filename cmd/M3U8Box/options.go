package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sydove/M3U8Box/internal/utils"
)

var errShowUsage = errors.New("show usage")

type runMode string

const (
	modeDownload runMode = "download"
	modePackage  runMode = "package"
)

type runOptions struct {
	mode        runMode
	links       []string
	dir         string
	name        string
	concurrency int
	mp4Path     string
	hlsTime     int
}

func parseRunOptions() (*runOptions, error) {
	var (
		url         string
		dir         string
		name        string
		concurrency int
		file        string
		mp4Path     string
		hlsTime     int
	)

	flag.Usage = printUsage
	flag.StringVar(&url, "i", "", "访问的URL地址")
	flag.StringVar(&file, "f", "", "文件路径")
	flag.StringVar(&mp4Path, "mp4", "", "本地MP4文件路径，生成对应的HLS清单")
	flag.StringVar(&dir, "d", "", "保存的文件目录，默认当前目录")
	flag.StringVar(&name, "n", "", "保存的文件名，默认使用视频标题")
	flag.IntVar(&concurrency, "c", 10, "并发下载数，默认10")
	flag.IntVar(&hlsTime, "hls-time", 10, "HLS切片时长（秒），仅对 -mp4 生效")
	flag.Parse()

	if mp4Path != "" && (url != "" || file != "") {
		return nil, fmt.Errorf("-mp4 不能与 -i 或 -f 同时使用")
	}

	if url == "" && file == "" && mp4Path == "" {
		return nil, errShowUsage
	}

	targetDir, err := resolveOutputDir(dir)
	if err != nil {
		return nil, err
	}

	if mp4Path != "" {
		absMP4Path, err := resolveMP4Path(mp4Path)
		if err != nil {
			return nil, err
		}
		if hlsTime <= 0 {
			return nil, fmt.Errorf("-hls-time 必须大于 0")
		}
		return &runOptions{
			mode:    modePackage,
			dir:     targetDir,
			name:    name,
			mp4Path: absMP4Path,
			hlsTime: hlsTime,
		}, nil
	}

	links, err := resolveLinks(url, file)
	if err != nil {
		return nil, err
	}

	return &runOptions{
		mode:        modeDownload,
		links:       links,
		dir:         targetDir,
		name:        name,
		concurrency: concurrency,
	}, nil
}

func resolveLinks(url string, file string) ([]string, error) {
	if file == "" {
		return []string{url}, nil
	}

	if err := utils.EnsureDir(file, false); err != nil {
		return nil, fmt.Errorf("指定文件不存在: %s", file)
	}

	fileLinks, err := utils.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	links := make([]string, 0, len(fileLinks))
	for _, link := range fileLinks {
		link = strings.TrimSpace(link)
		if link != "" {
			links = append(links, link)
		}
	}

	if len(links) == 0 {
		return nil, fmt.Errorf("文件中没有可用链接: %s", file)
	}

	return links, nil
}

func resolveOutputDir(dir string) (string, error) {
	if dir == "" {
		return utils.GetProjectPath()
	}

	if err := utils.EnsureDir(dir, false); err != nil {
		return "", fmt.Errorf("指定保存目录不存在: %s", dir)
	}

	return dir, nil
}

func resolveMP4Path(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("指定 MP4 文件不存在: %s", path)
		}
		return "", fmt.Errorf("读取 MP4 文件失败: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("指定 MP4 路径不是文件: %s", path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("获取 MP4 绝对路径失败: %w", err)
	}
	return absPath, nil
}

func usageText() string {
	programName := filepath.Base(flag.CommandLine.Name())
	return fmt.Sprintf("使用方法:\n  %s -i=<URL> [-d=<目录>] [-n=<文件名>] [-c=<并发数>]\n  %s -f=<文件路径> [-d=<目录>] [-n=<文件名>] [-c=<并发数>]\n  %s -mp4=<文件路径> [-d=<目录>] [-n=<名称>] [-hls-time=<秒>]", programName, programName, programName)
}

func helpText() string {
	return fmt.Sprintf(`%s

模式:
  下载模式:
    %s -i=<URL> [-d=<目录>] [-n=<文件名>] [-c=<并发数>]
    %s -f=<文件路径> [-d=<目录>] [-n=<文件名>] [-c=<并发数>]

  生成模式:
    %s -mp4=<文件路径> [-d=<目录>] [-n=<名称>] [-hls-time=<秒>]

下载相关参数:
  -i string
        目标页面 URL
  -f string
        包含多个 URL 的文本文件路径
  -c int
        下载并发数，默认 10

生成相关参数:
  -mp4 string
        本地 MP4 文件路径，生成对应的 HLS 清单
  -hls-time int
        HLS 切片时长（秒），默认 10

通用参数:
  -d string
        保存的文件目录，默认当前目录
  -n string
        输出名称；下载模式默认使用页面标题，生成模式默认使用 MP4 文件名

规则:
  - 下载模式需要提供 -i 或 -f
  - 生成模式需要提供 -mp4
  - -mp4 不能与 -i、-f 同时使用`, usageText(), programName(), programName(), programName())
}

func printUsage() {
	fmt.Println(helpText())
}

func programName() string {
	return filepath.Base(flag.CommandLine.Name())
}
