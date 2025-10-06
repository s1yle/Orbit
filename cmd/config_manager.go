package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var keysPath string

// ConfigManager 管理应用程序配置的加载、保存和验证
type ConfigManager struct {
	configPath   string
	userConfig   *UserConfig
	mu           sync.RWMutex
	lastLoadTime time.Time
	autoSave     bool
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
		autoSave:   true,
	}
}

// LoadConfig 从文件加载配置，如果文件不存在则创建默认配置
func (cm *ConfigManager) LoadConfig() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查配置文件是否存在
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// 创建默认配置
		return cm.createDefaultConfig()
	}

	// 读取配置文件
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析JSON
	var config UserConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cm.validateConfig(&config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	cm.userConfig = &config
	cm.lastLoadTime = time.Now()

	logger.Infof("配置已从 %s 加载", cm.configPath)
	return nil
}

// SaveConfig 保存配置到文件
func (cm *ConfigManager) SaveConfig() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.userConfig == nil {
		return fmt.Errorf("没有配置可保存")
	}

	// 更新最后更新时间
	cm.userConfig.LastUpdate = time.Now().Format("2006-01-02 15:04:05")

	// 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(cm.userConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	logger.Infof("配置已保存到 %s", cm.configPath)
	return nil
}

// GetConfig 获取当前配置的副本
func (cm *ConfigManager) GetConfig() *UserConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.userConfig == nil {
		return nil
	}

	// 返回配置的深拷贝
	return cm.deepCopyConfig(cm.userConfig)
}

// UpdateConfig 更新配置并自动保存
func (cm *ConfigManager) UpdateConfig(updater func(*UserConfig)) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.userConfig == nil {
		return fmt.Errorf("配置未加载")
	}

	// 创建配置副本进行修改
	newConfig := cm.deepCopyConfig(cm.userConfig)
	updater(newConfig)

	// 验证新配置
	if err := cm.validateConfig(newConfig); err != nil {
		return fmt.Errorf("配置更新验证失败: %w", err)
	}

	// 更新配置
	cm.userConfig = newConfig

	// 自动保存
	if cm.autoSave {
		return cm.saveConfigLocked()
	}

	return nil
}

// GetVSCodeConfig 获取VSCode配置
func (cm *ConfigManager) GetVSCodeConfig() *VSCodeConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.userConfig == nil {
		return nil
	}
	return &cm.userConfig.VSCode
}

// GetSoftwareConfig 获取软件配置
func (cm *ConfigManager) GetSoftwareConfig() *SoftwareConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.userConfig == nil {
		return nil
	}
	return &cm.userConfig.Software
}

// GetEncryptionConfig 获取加密配置
func (cm *ConfigManager) GetEncryptionConfig() *EncryptionConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.userConfig == nil {
		return nil
	}
	return &cm.userConfig.Encryption
}

// GetSystemConfig 获取系统配置
func (cm *ConfigManager) GetSystemConfig() *SystemConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.userConfig == nil {
		return nil
	}
	return &cm.userConfig.System
}

// UpdateSystemConfig 更新系统配置（如备份计数）
func (cm *ConfigManager) UpdateSystemConfig(updater func(*SystemConfig)) error {
	return cm.UpdateConfig(func(config *UserConfig) {
		updater(&config.System)
	})
}

// IsConfigLoaded 检查配置是否已加载
func (cm *ConfigManager) IsConfigLoaded() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.userConfig != nil
}

// GetConfigAge 获取配置的年龄
func (cm *ConfigManager) GetConfigAge() time.Duration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return time.Since(cm.lastLoadTime)
}

// ReloadConfig 重新加载配置
func (cm *ConfigManager) ReloadConfig() error {
	return cm.LoadConfig()
}

// SetAutoSave 设置是否自动保存
func (cm *ConfigManager) SetAutoSave(autoSave bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.autoSave = autoSave
}

// 内部方法

// createDefaultConfig 创建默认配置
func (cm *ConfigManager) createDefaultConfig() error {
	// 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 创建默认配置
	defaultConfig := cm.getDefaultConfig()

	// 序列化并保存
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化默认配置失败: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入默认配置文件失败: %w", err)
	}

	cm.userConfig = defaultConfig
	cm.lastLoadTime = time.Now()

	logger.Infof("默认配置已创建并保存到 %s", cm.configPath)
	return nil
}

// getDefaultConfig 返回默认配置
func (cm *ConfigManager) getDefaultConfig() *UserConfig {
	keysPath = filepath.Join(filepath.Dir(cm.configPath), "keys")

	return &UserConfig{
		System: SystemConfig{
			LastBackupTime:    "",
			BackupCount:       0,
			DefaultBackupPath: CurrentDir,
		},
		VSCode: VSCodeConfig{
			ConfigDirs: []ConfigDirType{
				{
					Name:         "APPDATA",
					Path:         CodeConfigDir,
					OriginalPath: filepath.Join(os.Getenv("APPDATA"), "Code"),
				},
				{
					Name:         "USER",
					Path:         CodeUserDir,
					OriginalPath: filepath.Join(os.Getenv("USERPROFILE"), ".vscode"),
				},
			},
			ExcludedExtensions: []string{},
			BackupSetting:      true,
		},
		Software: SoftwareConfig{
			ExcludedPatterns: []string{
				"Mozilla Firefox",
				"Visual Studio Code",
				"Google Chrome",
			},
			IncludeStoreApps: false,
			AutoUpdateList:   true,
		},
		Encryption: EncryptionConfig{
			Enabled:          true,
			PublicKeyPath:    filepath.Join(keysPath, "public.pem"),
			PrivateKeyPath:   filepath.Join(keysPath, "private.pem"),
			DefaultAlgorithm: "RSA-2048",
		},
		LastUpdate: time.Now().Format("2006-01-02 15:04:05"),
	}
}

// validateConfig 验证配置的有效性
func (cm *ConfigManager) validateConfig(config *UserConfig) error {
	// 验证系统配置
	if config.System.DefaultBackupPath == "" {
		return fmt.Errorf("默认备份路径不能为空")
	}

	// 验证VSCode配置
	if len(config.VSCode.ConfigDirs) == 0 {
		return fmt.Errorf("VSCode配置目录不能为空")
	}

	// 验证加密配置
	if config.Encryption.Enabled {
		if config.Encryption.PublicKeyPath == "" {
			return fmt.Errorf("启用加密时公钥路径不能为空")
		}
		if config.Encryption.PrivateKeyPath == "" {
			return fmt.Errorf("启用加密时私钥路径不能为空")
		}
	}

	return nil
}

// deepCopyConfig 创建配置的深拷贝
func (cm *ConfigManager) deepCopyConfig(config *UserConfig) *UserConfig {
	// 使用JSON序列化和反序列化实现深拷贝
	data, err := json.Marshal(config)
	if err != nil {
		logger.Errorf("配置深拷贝序列化失败: %v", err)
		return nil
	}

	var copyConfig UserConfig
	if err := json.Unmarshal(data, &copyConfig); err != nil {
		logger.Errorf("配置深拷贝反序列化失败: %v", err)
		return nil
	}

	return &copyConfig
}

// saveConfigLocked 在已持有锁的情况下保存配置
func (cm *ConfigManager) saveConfigLocked() error {
	if cm.userConfig == nil {
		return fmt.Errorf("没有配置可保存")
	}

	// 更新最后更新时间
	cm.userConfig.LastUpdate = time.Now().Format("2006-01-02 15:04:05")

	// 序列化配置
	data, err := json.MarshalIndent(cm.userConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// 全局配置管理器实例
var globalConfigManager *ConfigManager

// InitGlobalConfigManager 初始化全局配置管理器
func InitGlobalConfigManager() error {
	configPath := filepath.Join(os.Getenv("APPDATA"), "orbit_user", "info.json")
	globalConfigManager = NewConfigManager(configPath)
	return globalConfigManager.LoadConfig()
}

// GetConfigManager 获取全局配置管理器
func GetConfigManager() *ConfigManager {
	return globalConfigManager
}
