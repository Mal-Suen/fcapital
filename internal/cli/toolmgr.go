package cli

import (
	"sync"

	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
)

var (
	toolManager     *toolmgr.ToolManager
	toolManagerOnce sync.Once
)

// GetToolManager 获取全局工具管理器实例
func GetToolManager() *toolmgr.ToolManager {
	toolManagerOnce.Do(func() {
		toolManager = toolmgr.NewToolManager()
		// 加载工具配置
		if err := toolManager.LoadToolsFromYAML("configs/tools.yaml"); err != nil {
			// 如果加载失败，使用默认配置
			toolManager.LoadToolsFromYAML("")
		}
	})
	return toolManager
}

// InitToolManager 初始化工具管理器并检测所有工具
func InitToolManager() *toolmgr.ToolManager {
	tm := GetToolManager()
	tm.DetectAll()
	return tm
}
