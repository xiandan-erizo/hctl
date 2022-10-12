// CheckAndTransformFilePath return converted path
package cmd

import (
	"os"
	"path/filepath"
	"strings"
)

func CheckAndTransformFilePath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir(), path[2:])
	}
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return "", err
	}
	return path, nil
}
