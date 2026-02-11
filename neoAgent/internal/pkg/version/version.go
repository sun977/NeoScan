// ### 发布流程
// 1. **更新版本号**：修改 `internal/pkg/version/version.go`
// 2. **运行发布脚本**：自动完成发布流程
// 3. **推送代码和 Tag**：推送到远程仓库
// 4. **验证构建**：测试各个平台的二进制文件
// 5. **更新文档**：更新 CHANGELOG 和 README

package version

var (
	Version    = "2.11.0" // 版本号 -- 发布时候更新版本号
	APIVersion = "2.0"
	BuildTime  string
	GitCommit  string
	GoVersion  string
)

func GetVersion() string {
	return Version
}

func GetFullVersion() string {
	return Version
}

func GetUserAgent() string {
	return "Mozilla/5.0 (compatible; NeoScan-Agent/" + Version + "; +https://github.com/sun977/NeoScan) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
}
