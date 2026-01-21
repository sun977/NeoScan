# NeoAgent CLI 开发指南

本文档旨在帮助开发者快速定位命令源码，并指导如何修改命令提示信息（Help）和参数（Flags）。

## 1. 命令结构与源码映射

NeoAgent 使用 [Cobra](https://github.com/spf13/cobra) 框架，目录结构严格对应 CLI 命令结构。

| CLI 命令 | 源码文件路径 | 说明 |
| :--- | :--- | :--- |
| **`neoAgent`** (Root) | [`cmd/agent/root.go`](../cmd/agent/root.go) | 根命令，定义全局 Flag (如 `--config`) |
| **`neoAgent server`** | [`cmd/agent/server.go`](../cmd/agent/server.go) | 启动 Worker 服务模式 |
| **`neoAgent proxy`** | [`cmd/agent/proxy/root.go`](../cmd/agent/proxy/root.go) | 代理/转发服务 |
| **`neoAgent scan`** | [`cmd/agent/scan/root.go`](../cmd/agent/scan/root.go) | 扫描父命令 |
| `neoAgent scan asset` | [`cmd/agent/scan/asset.go`](../cmd/agent/scan/asset.go) | 资产发现扫描 |
| `neoAgent scan port` | [`cmd/agent/scan/port.go`](../cmd/agent/scan/port.go) | 端口扫描 |
| `neoAgent scan web` | [`cmd/agent/scan/web.go`](../cmd/agent/scan/web.go) | Web 综合扫描 |
| `neoAgent scan dir` | [`cmd/agent/scan/dir.go`](../cmd/agent/scan/dir.go) | 目录扫描 |
| `neoAgent scan subdomain` | [`cmd/agent/scan/subdomain.go`](../cmd/agent/scan/subdomain.go) | 子域名扫描 |
| `neoAgent scan vuln` | [`cmd/agent/scan/vuln.go`](../cmd/agent/scan/vuln.go) | 漏洞扫描 (Nuclei) |

---

## 2. 如何修改提示信息

每个命令文件（如 `asset.go`）中都有一个 `cobra.Command` 结构体定义。修改其中的字段即可更新帮助信息。

**示例：修改资产扫描描述**
打开 [`cmd/agent/scan/asset.go`](../cmd/agent/scan/asset.go)：

```go
var cmd = &cobra.Command{
    Use:   "asset",
    Short: "资产发现扫描",  // <--- 修改这里：简短描述 (neoAgent scan --help 显示)
    Long:  `对指定网段或 IP 进行资产存活探测、端口开放检测及指纹识别。
支持 ICMP Ping 探测和 TCP SYN 扫描...`, // <--- 修改这里：详细描述 (neoAgent scan asset --help 显示)
    // ...
}
```

---

## 3. 如何修改/添加命令参数 (Flags)

NeoAgent 采用了 **Options 模式** 来解耦 CLI 和核心逻辑。修改参数需要遵循 **3步走** 流程。

### 步骤 1: 修改 Options 结构体定义
> 位置：`internal/core/options/`

打开对应的 Options 文件（例如 [`internal/core/options/scan_asset.go`](../internal/core/options/scan_asset.go)）。

1.  **在结构体中添加字段**：
    ```go
    type AssetScanOptions struct {
        Target   string
        Port     string
        Rate     int
        Timeout  int  // <--- 新增字段
    }
    ```
2.  **更新 `New...Options` 默认值**（可选）：
    ```go
    func NewAssetScanOptions() *AssetScanOptions {
        return &AssetScanOptions{
            Timeout: 10, // <--- 设置默认值
        }
    }
    ```
3.  **更新 `ToTask` 映射逻辑**：
    ```go
    func (o *AssetScanOptions) ToTask() *model.Task {
        // ...
        task.Params["timeout"] = o.Timeout // <--- 映射到核心 Task 参数
        return task
    }
    ```

### 步骤 2: 绑定 CLI Flag
> 位置：`cmd/agent/scan/`

打开对应的 Command 文件（例如 [`cmd/agent/scan/asset.go`](../cmd/agent/scan/asset.go)）。

在 `NewAssetScanCmd` 函数中绑定新的 Flag：

```go
func NewAssetScanCmd() *cobra.Command {
    // ...
    flags := cmd.Flags()
    // ... 原有 flags
    
    // <--- 新增绑定
    // 参数说明: &变量地址, 长参数名, 默认值, 帮助说明
    flags.IntVar(&opts.Timeout, "timeout", opts.Timeout, "扫描超时时间(秒)")
    
    return cmd
}
```

### 步骤 3: 重新编译验证
在 `neoAgent` 目录下执行：
```powershell
go build -o neoAgent.exe ./cmd/agent
./neoAgent.exe scan asset --help
```
你将看到新添加的 `--timeout` 参数。

---

## 4. 核心参数文件索引

所有参数定义的源头都在 `internal/core/options` 目录下：

| 命令 | Options 文件 |
| :--- | :--- |
| `proxy` | [`internal/core/options/proxy.go`](../internal/core/options/proxy.go) |
| `scan asset` | [`internal/core/options/scan_asset.go`](../internal/core/options/scan_asset.go) |
| `scan port` | [`internal/core/options/scan_port.go`](../internal/core/options/scan_port.go) |
| `scan web` | [`internal/core/options/scan_web.go`](../internal/core/options/scan_web.go) |
| `scan dir` | [`internal/core/options/scan_dir.go`](../internal/core/options/scan_dir.go) |
| `scan subdomain` | [`internal/core/options/scan_subdomain.go`](../internal/core/options/scan_subdomain.go) |
| `scan vuln` | [`internal/core/options/scan_vuln.go`](../internal/core/options/scan_vuln.go) |
