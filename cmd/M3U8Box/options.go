package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sydove/M3U8Box/internal/utils"
)

var errShowUsage = errors.New("show usage")

type runOptions struct {
	links       []string
	dir         string
	name        string
	concurrency int
}

func parseRunOptions() (*runOptions, error) {
	var (
		url         string
		dir         string
		name        string
		concurrency int
		file        string
	)

	flag.Usage = printUsage
	flag.StringVar(&url, "i", "", "访问的URL地址")
	flag.StringVar(&file, "f", "", "文件路径")
	flag.StringVar(&dir, "d", "", "保存的文件目录，默认当前目录")
	flag.StringVar(&name, "n", "", "保存的文件名，默认使用视频标题")
	flag.IntVar(&concurrency, "c", 10, "并发下载数，默认10")
	flag.Parse()

	if url == "" && file == "" {
		return nil, errShowUsage
	}

	links, err := resolveLinks(url, file)
	if err != nil {
		return nil, err
	}

	targetDir, err := resolveOutputDir(dir)
	if err != nil {
		return nil, err
	}

	return &runOptions{
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

func usageText() string {
	programName := filepath.Base(flag.CommandLine.Name())
	return fmt.Sprintf("使用方法:\n  %s -i=<URL> [-d=<目录>] [-n=<文件名>] [-c=<并发数>]\n  %s -f=<文件路径> [-d=<目录>] [-n=<文件名>] [-c=<并发数>]", programName, programName)
}

func helpText() string {
	var buf bytes.Buffer
	oldOutput := flag.CommandLine.Output()
	flag.CommandLine.SetOutput(&buf)
	flag.PrintDefaults()
	flag.CommandLine.SetOutput(oldOutput)

	return fmt.Sprintf("%s\n\n选项:\n%s", usageText(), strings.TrimRight(buf.String(), "\n"))
}

func printUsage() {
	fmt.Println(helpText())
}
