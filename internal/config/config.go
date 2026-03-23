package config

import (
	"ic-wails/internal/untils"
	"os"
	"path"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/creasty/defaults"
)

var (
	instance *ApplicationConfigManager
	mutex    sync.Mutex
)

type ApplicationConfigManager struct {
	mutex     sync.Mutex
	AppConfig *AppConfig
}

// NewApplicationConfigManager 创建应用配置管理器单例。
func NewApplicationConfigManager() *ApplicationConfigManager {
	if instance != nil {
		return instance
	}
	mutex.Lock()
	defer mutex.Unlock()
	instance = &ApplicationConfigManager{mutex: sync.Mutex{}}
	instance.AppConfig = instance.LoadAppConfig()
	return instance
}

// SaveAppConfig 保存配置并刷新
func (acm *ApplicationConfigManager) SaveAppConfig(config AppConfig) {
	acm.saveAppConfig(config)
	acm.ReloadAppConfig()
}

// 保存配置
func (acm *ApplicationConfigManager) saveAppConfig(config AppConfig) {
	configPath := getConfigPath()
	tomlBytes, err := toml.Marshal(config)
	if err != nil {
		panic(err)
	}
	writeErr := os.WriteFile(configPath, tomlBytes, 0644)
	if writeErr != nil {
		panic(writeErr)
	}
}

// LoadAppConfig 加载配置
func (acm *ApplicationConfigManager) LoadAppConfig() *AppConfig {
	config := &AppConfig{}
	configPath := getConfigPath()
	// 如果没有配置文件就按照默认值生成一份，并写入文件中
	if _, statError := os.Stat(configPath); os.IsNotExist(statError) {
		initError := defaults.Set(config)
		if initError != nil {
			panic(initError)
		}
		acm.saveAppConfig(*config)
		return config
	}
	appConfigFile, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	appConfigStr := string(appConfigFile)
	_, err = toml.Decode(appConfigStr, config)
	return config
}

// ReloadAppConfig 重新加载配置文件信息,会刷新当前实例配置字段
func (acm *ApplicationConfigManager) ReloadAppConfig() *AppConfig {
	mutex.Lock()
	defer mutex.Unlock()
	config := acm.LoadAppConfig()
	acm.AppConfig = config
	return config
}

// getConfigPath 返回应用配置文件所在路径。
func getConfigPath() string {
	dir := untils.RunningDir()
	return path.Join(dir, "config.toml")
}

type AppConfig struct {
	App    AppMeta      `toml:"app"`
	Window WindowConfig `toml:"window"`
	Ai     AiConfig     `toml:"ai"`
}

type AiConfig struct {
	ApiKey  string `toml:"api_key"`
	Model   string `toml:"model" default:"doubao-seed-1-6-250615"`
	BaseUrl string `toml:"base_url" default:"https://ark.cn-beijing.volces.com/api/v3"`
}

type AppMeta struct {
	Name    string `toml:"name" default:"ic-wails"`
	Version string `toml:"version" default:"Alpha 0.0.1.0"`
}

type WindowConfig struct {
	Width  int    `toml:"width" default:"1024"`
	Height int    `toml:"height" default:"768"`
	X      string `toml:"x"`
	Y      string `toml:"y"`
}

// GetWidth 返回满足窗口约束的宽度，非法值时回退默认值。
func (wc *WindowConfig) GetWidth() int {
	defaultWidth := 800
	width := wc.Width
	if width >= 800 && width <= 3840 {
		return width
	}
	return defaultWidth
}

// GetHeight 返回满足窗口约束的高度，非法值时回退默认值。
func (wc *WindowConfig) GetHeight() int {
	defHeight := 600
	height := wc.Height
	if height >= 600 && height <= 2160 {
		return height
	}
	return defHeight
}
