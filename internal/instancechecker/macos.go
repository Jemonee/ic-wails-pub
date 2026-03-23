//go:build darwin

package instancechecker

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

type MacosChecker struct {
	listener   net.Listener
	socketPath string
}

// NewMacosChecker 创建并初始化检查器
func NewMacosChecker(appName string) *MacosChecker {
	return &MacosChecker{
		socketPath: getSocketPath(appName),
	}
}

// IsAlreadyRunning 检查应用是否已经在运行
func (mc *MacosChecker) IsAlreadyRunning() bool {
	if err := os.MkdirAll(filepath.Dir(mc.socketPath), 0755); err != nil {
		fmt.Printf("Failed to create socket directory: %v\n", err)
		return false
	}

	// 尝试连接到现有的 Unix Socket
	conn, err := net.Dial("unix", mc.socketPath)
	if err == nil {
		// 连接成功，说明已有实例在运行
		conn.Close()
		return true
	}

	// 清理可能存在的旧socket文件
	os.Remove(mc.socketPath)

	// 创建新的监听器
	listener, err := net.Listen("unix", mc.socketPath)
	if err != nil {
		// 创建失败，可能权限问题或其他错误
		fmt.Printf("Failed to create socket: %v\n", err)
		return false
	}

	mc.listener = listener

	// 启动goroutine处理连接（保持socket活跃）
	go mc.handleConnections()

	return false
}

// handleConnections 处理其他实例的连接尝试
func (mc *MacosChecker) handleConnections() {
	for {
		conn, err := mc.listener.Accept()
		if err != nil {
			// 监听器已关闭，正常退出
			return
		}
		conn.Close()
	}
}

// Close 清理资源
func (mc *MacosChecker) Close() {
	if mc.listener != nil {
		mc.listener.Close()
	}
	os.Remove(mc.socketPath)
}

// getSocketPath 生成唯一的socket文件路径
func getSocketPath(appName string) string {
	// 使用用户主目录避免权限问题
	homeDir, _ := os.UserHomeDir()

	// 包含用户ID确保多用户不冲突
	uid := os.Getuid()

	// 生成唯一的socket文件路径
	socketName := fmt.Sprintf("%s-%d.sock", appName, uid)
	return filepath.Join(homeDir, ".config", appName, socketName)
}

func init() {
	instance = NewMacosChecker("ic-wails")
}
