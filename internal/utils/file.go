package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnsureDir 确保目录存在
func EnsureDir(dir string, create bool) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if create {
			return os.MkdirAll(dir, 0755)
		} else {
			return fmt.Errorf("目录不存在")
		}
	}
	return nil
}

func ReadFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	return strings.Split(string(data), "\n"), nil
}

// GetAbsPath 获取绝对路径
func GetAbsPath(path string) (string, error) {
	return filepath.Abs(path)
}

// GetProjectPath 获取当前运行时的绝对路径
func GetProjectPath() (string, error) {
	return filepath.Abs(".")
}

func GetTaskHash(title string) (string, error) {
	timeStr := time.Now().Format("150405")
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s", title, timeStr)))
	hashStr := hex.EncodeToString(hash[:])
	return hashStr, nil
}
