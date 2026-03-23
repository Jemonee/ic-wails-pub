package api

import (
	pkgapi "ic-wails/pkg/api"
	"ic-wails/pkg/common"
	"os/exec"
	"runtime"
	"strings"
)

type WindowManagerApi struct {
}

func (wma *WindowManagerApi) DoubleClickTitleBar() (result common.R[string]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	var res *string
	goos := runtime.GOOS
	if goos == "darwin" {
		action := wma.getTitleBarDoubleClickAction()
		res = &action
	}
	return common.S(res)
}

func (wma *WindowManagerApi) getTitleBarDoubleClickAction() string {
	if runtime.GOOS != "darwin" {
		return "NotSupported"
	}

	// 使用 defaults 命令读取系统设置
	cmd := exec.Command("defaults", "read", "NSGlobalDomain", "AppleActionOnDoubleClick")
	output, err := cmd.Output()
	if err != nil {
		return "Maximize" // 默认值
	}

	return strings.TrimSpace(string(output))
}

func NewWindowManagerApi() *WindowManagerApi {
	return &WindowManagerApi{}
}
