package merge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sydove/M3U8Box/internal/logger"
)

type Merger interface {
	Merge(m3u8File string, videoPath string) error
	Modify(m3u8File, cryptPath string, staticPath string, tsPaths []string) (string, error)
}

type FmgMerger struct{}

func (f *FmgMerger) Merge(m3u8File string, videoPath string) error {
	logger.Infof("下载完成,开始合成视频!")
	args := []string{
		"-allowed_extensions", "ALL",
		"-protocol_whitelist", "file,http,https,tcp,tls,crypto",
		"-i", m3u8File,
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
		videoPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	// ffmpeg 输出仅写入日志文件，避免直接打印到终端
	cmd.Stdout = logger.FileWriter()
	cmd.Stderr = logger.FileWriter()

	if err := cmd.Run(); err != nil {
		logger.Errorf("执行 ffmpeg 命令失败: %v", err)
		return err
	}

	logger.Infof("视频整合完成: %s", videoPath)
	return nil
}

func (f *FmgMerger) Modify(m3u8File string, cryptPath string, staticPath string, tsPaths []string) (string, error) {
	// 读取原始 m3u8 文件
	content, err := os.ReadFile(m3u8File)
	if err != nil {
		return "", err
	}
	m3u8Content := string(content)

	// 替换 crypt.key 路径
	if cryptPath != "" {
		relativeCryptPath, err := filepath.Rel(staticPath, cryptPath)
		if err != nil {
			return "", fmt.Errorf("生成相对密钥路径失败: %w", err)
		}
		// 使用正则表达式替换 URI 中的密钥路径
		cryptRule := regexp.MustCompile(`URI="[^"]+"`)
		m3u8Content = cryptRule.ReplaceAllString(m3u8Content, fmt.Sprintf(`URI="%s"`, filepath.ToSlash(relativeCryptPath)))
	}

	// 替换 TS 文件路径
	if len(tsPaths) > 0 {
		// 提取原始 TS 文件路径模式
		tsRule := regexp.MustCompile(`https?://[^\n]+\.ts[^\n]*`)
		tsMatches := tsRule.FindAllString(m3u8Content, -1)

		// 替换每个 TS 文件路径
		for i, tsUrl := range tsMatches {
			if i < len(tsPaths) {
				relativeTSPath, err := filepath.Rel(staticPath, tsPaths[i])
				if err != nil {
					return "", fmt.Errorf("生成相对 ts 路径失败: %w", err)
				}
				m3u8Content = strings.Replace(m3u8Content, tsUrl, filepath.ToSlash(relativeTSPath), 1)
			}
		}
	}

	modifiedM3U8Path := filepath.Join(staticPath, "modified.m3u8")
	err = os.WriteFile(modifiedM3U8Path, []byte(m3u8Content), 0644)
	if err != nil {
		logger.Errorf("保存修改后的 m3u8 文件失败: %v", err)
		return "", err
	}
	return modifiedM3U8Path, nil
}
