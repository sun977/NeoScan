package core

// CommandBuilder 定义命令构建接口
// 它的职责是将抽象的配置转换为具体的操作系统命令
type CommandBuilder interface {
	// Build 根据目标和配置生成可执行的命令字符串和参数列表
	// 返回: (命令路径, 参数列表, 错误)
	Build(target string, config map[string]interface{}) (string, []string, error)
}
