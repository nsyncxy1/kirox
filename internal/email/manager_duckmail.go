package email

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"reg_go/internal/storage"
)

// getDuckMailConfigPath 获取 DuckMail 配置文件路径
func getDuckMailConfigPath() string {
	return filepath.Join(storage.GetDataDir(), "duckmail.dat")
}

// GetDuckMailConfigs 读取已保存的 DuckMail 配置列表
func GetDuckMailConfigs() []DuckMailConfig {
	data, err := os.ReadFile(getDuckMailConfigPath())
	if err != nil {
		return []DuckMailConfig{}
	}

	var configs []DuckMailConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		log.Printf("[DuckMail] 配置文件格式无效，已重置: %v", err)
		os.Remove(getDuckMailConfigPath())
		return []DuckMailConfig{}
	}

	return configs
}

// SaveDuckMailConfigs 保存 DuckMail 配置列表（JSON 字符串输入）
func SaveDuckMailConfigs(configsJSON string) map[string]interface{} {
	var configs []DuckMailConfig
	if err := json.Unmarshal([]byte(configsJSON), &configs); err != nil {
		return map[string]interface{}{"error": "配置格式错误: " + err.Error()}
	}

	// 基本验证
	for i, cfg := range configs {
		if cfg.Name == "" {
			return map[string]interface{}{"error": fmt.Sprintf("第 %d 个配置缺少名称", i+1)}
		}
		if cfg.APIURL == "" {
			return map[string]interface{}{"error": fmt.Sprintf("配置 %s 缺少 API URL", cfg.Name)}
		}
		if cfg.Domain == "" {
			return map[string]interface{}{"error": fmt.Sprintf("配置 %s 缺少域名", cfg.Name)}
		}
	}

	jsonData, _ := json.Marshal(configs)
	os.MkdirAll(filepath.Dir(getDuckMailConfigPath()), 0755)
	if err := os.WriteFile(getDuckMailConfigPath(), jsonData, 0600); err != nil {
		return map[string]interface{}{"error": "保存失败: " + err.Error()}
	}

	log.Printf("[DuckMail] 已保存 %d 个配置", len(configs))
	return map[string]interface{}{"success": true}
}

// TestDuckMailConnection 测试 DuckMail 连接（JSON 字符串输入单个配置）
func TestDuckMailConnection(configJSON string) map[string]interface{} {
	var config DuckMailConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return map[string]interface{}{"error": "配置格式错误: " + err.Error()}
	}

	client := NewDuckMailClient(config)
	domains, err := client.TestConnection()
	if err != nil {
		return map[string]interface{}{"error": "连接失败: " + err.Error()}
	}

	return map[string]interface{}{
		"success":     true,
		"domains":     domains,
		"domainCount": len(domains),
	}
}
