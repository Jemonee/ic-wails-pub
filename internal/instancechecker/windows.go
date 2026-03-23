//go:build windows

package instancechecker

import (
	"crypto/md5"
	"encoding/hex"
	"golang.org/x/sys/windows"
	"log"
	"os"
	"path/filepath"
)

const mutexName = "Global\\IC_EICTH_APPLICATION_A"

type windowsChecker struct {
	mutexHandle windows.Handle
}

func getMutexName() string {
	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	dir := filepath.Dir(executable)
	hasher := md5.New()
	hasher.Write([]byte(dir))
	sum := hasher.Sum(nil)
	mutexNameKey := mutexName + "_" + hex.EncodeToString(sum)
	log.Printf("当前MutexNameKey：%s", mutexNameKey)
	return mutexNameKey
}

func (c *windowsChecker) IsAlreadyRunning() bool {
	mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(getMutexName()))
	if err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			return true
		}
		log.Printf("Unexpected error creating mutex: %v", err)
		return false
	}
	c.mutexHandle = mutex
	return false
}

func (c *windowsChecker) Close() {
	if c.mutexHandle != 0 {
		_ = windows.CloseHandle(c.mutexHandle)
	}
}

func init() {
	instance = &windowsChecker{}
}
