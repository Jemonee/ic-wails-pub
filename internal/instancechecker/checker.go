package instancechecker

import (
	"os"
	"path/filepath"
)

type InstanceChecker interface {
	// IsAlreadyRunning 判断当前app是否已经启动
	IsAlreadyRunning() bool
	// Close app关闭的时候将启动时监测app是否启动的系统注入给取消掉
	Close()
}

var instance InstanceChecker

func GetInstanceChecker() InstanceChecker {
	return instance
}

func GetRunningDir() string {
	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(executable)
}
