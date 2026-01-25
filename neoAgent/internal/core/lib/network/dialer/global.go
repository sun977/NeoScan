package dialer

import (
	"time"
)

// GlobalDialer 全局拨号器实例
// 为了方便各个 Scanner 使用，这里使用单例模式，但支持 Set 方法替换
var globalDialer Dialer = NewDefaultDialer(3 * time.Second)

// SetGlobalDialer 设置全局拨号器 (例如配置了全局代理时)
func SetGlobalDialer(d Dialer) {
	globalDialer = d
}

// Get 获取全局拨号器
func Get() Dialer {
	return globalDialer
}
