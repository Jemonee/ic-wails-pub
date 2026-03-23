package untils

import (
	"os"
	"path/filepath"
)

// RunningDir 获取当前运行目录
func RunningDir() string {
	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(executable)
}
