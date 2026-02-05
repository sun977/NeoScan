# Pipeline 后期升级规划 - 内置模板 + 外部模板加载

## 一、设计理念

### 核心原则

**"简单实用，避免过度设计，零配置文件依赖"**

Agent 端的 Pipeline 应该：
- ✅ **简单线性流程**：Alive → Port → Service → OS → Web → Vuln
- ✅ **条件智能跳过**：根据前序结果决定是否执行后续阶段
- ✅ **内置预设模板**：常用扫描流程硬编码到代码中，无需配置文件
- ✅ **外部模板支持**：通过 `--template` 参数加载用户自定义 YAML 模板
- ✅ **模板样板生成**：通过 `--create` 参数生成模板样板，帮助用户理解模板规则
- ❌ **不使用复杂 DAG**：Master 端已经实现了 DAG 编排，Agent 只负责执行
- ❌ **不依赖配置文件**：二进制文件独立运行，无需携带配置目录

### 与原版方案的区别

| 维度 | 原版方案 | 新方案（本规划） |
|------|----------|------------------|
| **复杂度** | 高（DAG、拓扑排序） | 低（线性流程 + 条件跳过） |
| **配置方式** | YAML 配置文件 | 内置模板 + 外部 YAML（可选） |
| **依赖管理** | DAG 图 | 简单依赖链 |
| **条件执行** | 复杂表达式 | 简单条件判断 |
| **实现成本** | 高（1500-2000 行） | 中（800-1000 行） |
| **适用场景** | 复杂编排 | 常用扫描流程 + 自定义扩展 |
| **部署方式** | 需要配置目录 | 单个二进制文件 |

---

## 二、架构设计

### 整体架构

```
┌─────────────────────────────────────────┐
│         CLI Layer (命令行接口)          │
│  --preset <name>                        │
│  --template <path>                      │
│  --create <path>                        │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      PresetManager                 │
│  (管理内置模板 + 外部模板）         │
│  - 内置模板（硬编码）                │
│  - 外部模板（YAML 加载）             │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│        AutoRunner                   │
│  (执行线性流程 + 条件跳过）          │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│         PipelineStage               │
│  (单个扫描阶段）                    │
└───────────────────────────────────────┘
```

### 核心组件

1. **PresetManager**：预设模板管理器（内置模板 + 外部模板加载）
2. **ExternalTemplateLoader**：外部 YAML 模板加载器
3. **TemplateGenerator**：模板样板生成器
4. **AutoRunner**：执行器（线性流程 + 条件跳过）
5. **PipelineStage**：阶段接口（支持条件判断）
6. **PipelineContext**：数据载体（在各阶段间传递）

---

## 三、预设模板设计

### 3.1 内置预设模板（硬编码）

内置模板直接硬编码在代码中，无需配置文件，提供 4 个常用扫描流程：

```go
package pipeline

import "time"

// builtinPresets 内置预设模板
var builtinPresets = map[string]*Preset{
    "full_scan": {
        Name:        "full_scan",
        Description: "完整的扫描流程：存活 → 端口 → 服务 → OS → Web → 漏洞",
        Timeout:     60 * time.Minute,
        Stages: []StageConfig{
            {
                Name:     "Alive Scan",
                Type:     "alive",
                Critical: true,
                Params: map[string]interface{}{
                    "concurrency":     1000,
                    "resolve_hostname": true,
                },
            },
            {
                Name:         "Port Scan",
                Type:         "port",
                Critical:     true,
                DependsOn:    []string{"Alive Scan"},
                Params: map[string]interface{}{
                    "port_range":     "1-65535",
                    "rate":          1000,
                    "service_detect": false,
                },
            },
            {
                Name:          "Service Detect",
                Type:          "service",
                Critical:      false,
                DependsOn:     []string{"Port Scan"},
                SkipCondition: "no_open_ports",
                Params: map[string]interface{}{
                    "rate": 100,
                },
            },
            {
                Name:          "OS Fingerprint",
                Type:          "os",
                Critical:      false,
                DependsOn:     []string{"Service Detect"},
                SkipCondition: "no_open_ports",
                Params: map[string]interface{}{
                    "mode": "auto",
                },
            },
            {
                Name:          "Web Scan",
                Type:          "web",
                Critical:      false,
                DependsOn:     []string{"Service Detect"},
                SkipCondition: "no_http_service",
                Params: map[string]interface{}{
                    "concurrency": 10,
                    "depth":       3,
                },
            },
            {
                Name:          "Vuln Scan",
                Type:          "vuln",
                Critical:      false,
                DependsOn:     []string{"Port Scan"},
                SkipCondition: "no_open_ports",
                Params: map[string]interface{}{
                    "concurrency": 5,
                    "severity":     "high,critical",
                },
            },
        },
    },

    "quick_scan": {
        Name:        "quick_scan",
        Description: "快速扫描：存活 → 端口（前1000端口）",
        Timeout:     10 * time.Minute,
        Stages: []StageConfig{
            {
                Name:     "Alive Scan",
                Type:     "alive",
                Critical: true,
                Params: map[string]interface{}{
                    "concurrency": 1000,
                },
            },
            {
                Name:         "Port Scan",
                Type:         "port",
                Critical:     true,
                DependsOn:    []string{"Alive Scan"},
                Params: map[string]interface{}{
                    "port_range":     "1-1000",
                    "rate":          1000,
                    "service_detect": false,
                },
            },
        },
    },

    "web_scan": {
        Name:        "web_scan",
        Description: "Web 应用扫描：存活 → 端口 → 服务 → Web",
        Timeout:     30 * time.Minute,
        Stages: []StageConfig{
            {
                Name:     "Alive Scan",
                Type:     "alive",
                Critical: true,
                Params: map[string]interface{}{
                    "concurrency": 1000,
                },
            },
            {
                Name:         "Port Scan",
                Type:         "port",
                Critical:     true,
                DependsOn:    []string{"Alive Scan"},
                Params: map[string]interface{}{
                    "port_range":     "80,443,8080,8443",
                    "rate":          500,
                    "service_detect": false,
                },
            },
            {
                Name:          "Service Detect",
                Type:          "service",
                Critical:      false,
                DependsOn:     []string{"Port Scan"},
                SkipCondition: "no_open_ports",
                Params: map[string]interface{}{
                    "rate": 50,
                },
            },
            {
                Name:          "Web Scan",
                Type:          "web",
                Critical:      false,
                DependsOn:     []string{"Service Detect"},
                SkipCondition: "no_http_service",
                Params: map[string]interface{}{
                    "concurrency": 10,
                    "depth":       5,
                },
            },
        },
    },

    "vuln_scan": {
        Name:        "vuln_scan",
        Description: "漏洞扫描：存活 → 端口 → 漏洞",
        Timeout:     45 * time.Minute,
        Stages: []StageConfig{
            {
                Name:     "Alive Scan",
                Type:     "alive",
                Critical: true,
                Params: map[string]interface{}{
                    "concurrency": 1000,
                },
            },
            {
                Name:         "Port Scan",
                Type:         "port",
                Critical:     true,
                DependsOn:    []string{"Alive Scan"},
                Params: map[string]interface{}{
                    "port_range":     "top1000",
                    "rate":          1000,
                    "service_detect": false,
                },
            },
            {
                Name:          "Vuln Scan",
                Type:          "vuln",
                Critical:      false,
                DependsOn:     []string{"Port Scan"},
                SkipCondition: "no_open_ports",
                Params: map[string]interface{}{
                    "concurrency": 5,
                    "severity":     "medium,high,critical",
                },
            },
        },
    },
}
```

### 3.2 外部模板格式（YAML）

用户可以通过 `--template` 参数加载自定义 YAML 模板：

```yaml
# Pipeline 自定义模板
# 使用方法：neoAgent scan run -t 192.168.1.0/24 --template my_template.yaml

name: "my_custom_scan"
description: "我的自定义扫描流程"
timeout: 30m

stages:
  - name: "Alive Scan"
    type: "alive"
    critical: true
    params:
      concurrency: 1000
      resolve_hostname: true

  - name: "Port Scan"
    type: "port"
    critical: true
    depends_on:
      - "Alive Scan"
    params:
      port_range: "1-65535"
      rate: 1000
      service_detect: false

  - name: "Service Detect"
    type: "service"
    critical: false
    depends_on:
      - "Port Scan"
    skip_condition: "no_open_ports"
    params:
      rate: 100

  - name: "Web Scan"
    type: "web"
    critical: false
    depends_on:
      - "Service Detect"
    skip_condition: "no_http_service"
    params:
      concurrency: 10
      depth: 3
```

### 3.3 条件表达式定义

**支持的条件表达式**（简化版）：

| 表达式 | 含义 | 适用场景 |
|--------|------|----------|
| `no_open_ports` | 没有开放端口 | Service、OS、Vuln |
| `no_http_service` | 没有 HTTP 服务 | Web Scan |
| `no_ssh_service` | 没有 SSH 服务 | Brute Force |
| `no_db_service` | 没有数据库服务 | DB Vuln |
| `alive_failed` | 存活扫描失败 | 所有后续阶段 |

**实现方式**：在 `PipelineContext` 中添加辅助方法

```go
func (c *PipelineContext) shouldSkip(condition string) bool {
    switch condition {
    case "no_open_ports":
        return len(c.OpenPorts) == 0
    case "no_http_service":
        return !c.HasHTTPService
    case "no_ssh_service":
        return !c.HasSSHService
    case "no_db_service":
        return !c.HasDBService
    case "alive_failed":
        return !c.Alive
    default:
        return false
    }
}
```

---

## 四、CLI 命令设计

### 4.1 命令结构

```bash
# 使用内置预设模板
neoAgent scan run -t <target> --preset <preset_name> [options]

# 使用外部自定义模板
neoAgent scan run -t <target> --template <template_path> [options]

# 生成模板样板
neoAgent scan template create <output_path>

# 列出所有内置预设模板
neoAgent scan template list

# 查看内置预设模板详情
neoAgent scan template show <preset_name>
```

### 4.2 命令示例

```bash
# 使用内置全量扫描模板
./neoAgent scan run -t 192.168.1.0/24 --preset full_scan -c 100

# 使用内置快速扫描模板
./neoAgent scan run -t 192.168.1.0/24 --preset quick_scan -c 500

# 使用外部自定义模板
./neoAgent scan run -t 192.168.1.0/24 --template ./my_scan.yaml -c 100

# 生成模板样板
./neoAgent scan template create ./my_template.yaml

# 列出所有内置模板
./neoAgent scan template list

# 查看内置模板详情
./neoAgent scan template show full_scan
```

### 4.3 命令参数说明

| 参数 | 说明 | 示例 |
|------|------|------|
| `-t, --target` | 扫描目标（IP、CIDR、域名） | `-t 192.168.1.0/24` |
| `-c, --concurrency` | 并发数 | `-c 100` |
| `--preset` | 使用内置预设模板 | `--preset full_scan` |
| `--template` | 使用外部 YAML 模板 | `--template ./my.yaml` |
| `--create` | 生成模板样板 | `--create ./template.yaml` |
| `--list` | 列出所有内置模板 | `--list` |
| `--show` | 查看内置模板详情 | `--show full_scan` |

---

## 五、核心接口设计

### 5.1 PipelineStage 接口

```go
package pipeline

import (
    "context"
    "neoagent/internal/core/model"
)

// PipelineStage 扫描阶段接口
type PipelineStage interface {
    // 阶段名称
    Name() string

    // 阶段类型
    Type() model.TaskType

    // 执行阶段
    Execute(ctx context.Context, pCtx *PipelineContext) error

    // 是否应该跳过此阶段（条件判断）
    ShouldSkip(pCtx *PipelineContext) bool

    // 是否为关键阶段（关键阶段失败则终止）
    IsCritical() bool

    // 依赖的阶段（用于顺序执行）
    DependsOn() []string
}
```

### 5.2 PresetManager 接口

```go
package pipeline

// PresetManager 预设模板管理器
type PresetManager interface {
    // 获取所有内置预设模板
    ListBuiltinPresets() []PresetInfo

    // 获取内置预设模板详情
    GetBuiltinPreset(name string) (*Preset, error)

    // 加载外部模板
    LoadExternalTemplate(path string) (*Preset, error)

    // 验证预设模板
    ValidatePreset(preset *Preset) error
}

// PresetInfo 预设模板信息
type PresetInfo struct {
    Name        string
    Description string
    Timeout     time.Duration
    StageCount  int
    Source      string // "builtin" 或 "external"
}

// Preset 预设模板
type Preset struct {
    Name        string
    Description string
    Timeout     time.Duration
    Stages      []StageConfig
}

// StageConfig 阶段配置
type StageConfig struct {
    Name          string
    Type          string // "alive", "port", "service", "os", "web", "vuln"
    Critical      bool
    DependsOn     []string
    SkipCondition string
    Params        map[string]interface{}
}
```

### 5.3 ExternalTemplateLoader 接口

```go
package pipeline

// ExternalTemplateLoader 外部模板加载器
type ExternalTemplateLoader interface {
    // 从 YAML 文件加载模板
    LoadFromYAML(path string) (*Preset, error)

    // 验证模板格式
    ValidateTemplate(preset *Preset) error
}
```

### 5.4 TemplateGenerator 接口

```go
package pipeline

// TemplateGenerator 模板样板生成器
type TemplateGenerator interface {
    // 生成模板样板文件
    GenerateTemplate(path string) error

    // 生成模板样板内容
    GenerateTemplateContent() string
}
```

### 5.5 AutoRunner 接口

```go
package pipeline

// AutoRunner 自动化执行器
type AutoRunner interface {
    // 使用预设模板执行扫描
    RunWithPreset(ctx context.Context, presetName string, target string) ([]*PipelineContext, error)

    // 使用自定义配置执行扫描
    RunWithConfig(ctx context.Context, config *Preset, target string) ([]*PipelineContext, error)
}
```

---

## 六、实现方案

### 6.1 目录结构

```
internal/core/pipeline/
├── pipeline.go              # Pipeline 核心接口
├── stage.go                # Stage 接口和基础实现
├── context.go               # PipelineContext 数据载体
├── auto_runner.go           # AutoRunner 执行器
├── preset_manager.go        # PresetManager 预设模板管理器（内置 + 外部）
├── builtin_presets.go       # 内置预设模板（硬编码）
├── external_loader.go       # ExternalTemplateLoader 外部模板加载器
├── template_generator.go    # TemplateGenerator 模板样板生成器
├── stages/                 # 各个阶段的具体实现
│   ├── alive_stage.go
│   ├── port_stage.go
│   ├── service_stage.go
│   ├── os_stage.go
│   ├── web_stage.go
│   └── vuln_stage.go
└── README.md               # 本文档
```

### 6.2 PipelineContext 扩展

```go
package pipeline

import (
    "sync"

    "neoagent/internal/core/model"
)

// PipelineContext 扫描上下文
type PipelineContext struct {
    IP string

    // 阶段 1: 存活信息
    Alive    bool
    Hostname string
    OSGuess  string

    // 阶段 2: 端口开放信息
    OpenPorts []int

    // 阶段 3: 服务识别信息
    Services map[int]*model.PortServiceResult

    // 阶段 4: 操作系统精确识别
    OSInfo *model.OsInfo

    // 阶段 5: Web 扫描信息
    WebResults []*model.WebResult

    // 阶段 6: 漏洞扫描信息
    VulnResults []*model.VulnResult

    // 辅助字段：服务类型分类
    HasHTTPService bool
    HasSSHService   bool
    HasFTPServer   bool
    HasDBService    bool

    mu sync.RWMutex
}

// NewPipelineContext 创建 PipelineContext
func NewPipelineContext(ip string) *PipelineContext {
    return &PipelineContext{
        IP:       ip,
        Services: make(map[int]*model.PortServiceResult),
    }
}

// SetAlive 设置存活状态
func (c *PipelineContext) SetAlive(alive bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Alive = alive
}

// AddOpenPort 添加开放端口
func (c *PipelineContext) AddOpenPort(port int) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.OpenPorts = append(c.OpenPorts, port)
}

// SetService 设置服务信息
func (c *PipelineContext) SetService(port int, result *model.PortServiceResult) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Services[port] = result
    c.updateServiceTypes()
}

// SetOS 设置 OS 信息
func (c *PipelineContext) SetOS(info *model.OsInfo) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.OSInfo = info
}

// AddWebResult 添加 Web 扫描结果
func (c *PipelineContext) AddWebResult(result *model.WebResult) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.WebResults = append(c.WebResults, result)
}

// AddVulnResult 添加漏洞扫描结果
func (c *PipelineContext) AddVulnResult(result *model.VulnResult) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.VulnResults = append(c.VulnResults, result)
}

// updateServiceTypes 更新服务类型标记
func (c *PipelineContext) updateServiceTypes() {
    c.HasHTTPService = false
    c.HasSSHService = false
    c.HasFTPServer = false
    c.HasDBService = false

    for _, svc := range c.Services {
        switch svc.Service {
        case "http", "https", "http-alt", "http-proxy":
            c.HasHTTPService = true
        case "ssh":
            c.HasSSHService = true
        case "ftp", "ftp-data":
            c.HasFTPServer = true
        case "mysql", "postgresql", "mssql", "oracle", "mongodb":
            c.HasDBService = true
        }
    }
}

// shouldSkip 判断是否应该跳过某个阶段
func (c *PipelineContext) shouldSkip(condition string) bool {
    switch condition {
    case "no_open_ports":
        return len(c.OpenPorts) == 0
    case "no_http_service":
        return !c.HasHTTPService
    case "no_ssh_service":
        return !c.HasSSHService
    case "no_db_service":
        return !c.HasDBService
    case "alive_failed":
        return !c.Alive
    default:
        return false
    }
}
```

### 6.3 PresetManager 实现

```go
package pipeline

import (
    "fmt"
    "sync"
)

// presetManager 预设模板管理器
type presetManager struct {
    builtinPresets map[string]*Preset
    externalLoader ExternalTemplateLoader
    mu             sync.RWMutex
}

// NewPresetManager 创建预设模板管理器
func NewPresetManager() PresetManager {
    return &presetManager{
        builtinPresets: builtinPresets,
        externalLoader: NewExternalTemplateLoader(),
    }
}

// ListBuiltinPresets 获取所有内置预设模板
func (pm *presetManager) ListBuiltinPresets() []PresetInfo {
    pm.mu.RLock()
    defer pm.mu.RUnlock()

    var infos []PresetInfo
    for _, preset := range pm.builtinPresets {
        infos = append(infos, PresetInfo{
            Name:        preset.Name,
            Description: preset.Description,
            Timeout:     preset.Timeout,
            StageCount:  len(preset.Stages),
            Source:      "builtin",
        })
    }

    return infos
}

// GetBuiltinPreset 获取内置预设模板详情
func (pm *presetManager) GetBuiltinPreset(name string) (*Preset, error) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()

    preset, ok := pm.builtinPresets[name]
    if !ok {
        return nil, fmt.Errorf("builtin preset not found: %s", name)
    }

    return preset, nil
}

// LoadExternalTemplate 加载外部模板
func (pm *presetManager) LoadExternalTemplate(path string) (*Preset, error) {
    preset, err := pm.externalLoader.LoadFromYAML(path)
    if err != nil {
        return nil, fmt.Errorf("failed to load external template: %w", err)
    }

    // 验证模板
    if err := pm.ValidatePreset(preset); err != nil {
        return nil, fmt.Errorf("invalid external template: %w", err)
    }

    return preset, nil
}

// ValidatePreset 验证预设模板
func (pm *presetManager) ValidatePreset(preset *Preset) error {
    // 1. 检查阶段名称唯一性
    stageNames := make(map[string]bool)
    for _, stage := range preset.Stages {
        if stageNames[stage.Name] {
            return fmt.Errorf("duplicate stage name: %s", stage.Name)
        }
        stageNames[stage.Name] = true
    }

    // 2. 检查依赖关系
    for _, stage := range preset.Stages {
        for _, dep := range stage.DependsOn {
            if !stageNames[dep] {
                return fmt.Errorf("stage %s depends on non-existent stage: %s", stage.Name, dep)
            }
        }
    }

    // 3. 检查条件表达式
    for _, stage := range preset.Stages {
        if stage.SkipCondition != "" && !isValidCondition(stage.SkipCondition) {
            return fmt.Errorf("invalid skip condition: %s", stage.SkipCondition)
        }
    }

    return nil
}

// isValidCondition 检查条件表达式是否有效
func isValidCondition(condition string) bool {
    validConditions := map[string]bool{
        "no_open_ports":  true,
        "no_http_service": true,
        "no_ssh_service":  true,
        "no_db_service":   true,
        "alive_failed":    true,
    }

    return validConditions[condition]
}
```

### 6.4 ExternalTemplateLoader 实现

```go
package pipeline

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

// externalTemplateLoader 外部模板加载器
type externalTemplateLoader struct{}

// NewExternalTemplateLoader 创建外部模板加载器
func NewExternalTemplateLoader() ExternalTemplateLoader {
    return &externalTemplateLoader{}
}

// LoadFromYAML 从 YAML 文件加载模板
func (l *externalTemplateLoader) LoadFromYAML(path string) (*Preset, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read template file: %w", err)
    }

    var preset Preset
    if err := yaml.Unmarshal(data, &preset); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    return &preset, nil
}

// ValidateTemplate 验证模板格式
func (l *externalTemplateLoader) ValidateTemplate(preset *Preset) error {
    if preset.Name == "" {
        return fmt.Errorf("template name is required")
    }

    if len(preset.Stages) == 0 {
        return fmt.Errorf("template must have at least one stage")
    }

    for _, stage := range preset.Stages {
        if stage.Name == "" {
            return fmt.Errorf("stage name is required")
        }

        if stage.Type == "" {
            return fmt.Errorf("stage type is required")
        }

        if !isValidStageType(stage.Type) {
            return fmt.Errorf("invalid stage type: %s", stage.Type)
        }
    }

    return nil
}

// isValidStageType 检查阶段类型是否有效
func isValidStageType(stageType string) bool {
    validTypes := map[string]bool{
        "alive":   true,
        "port":    true,
        "service": true,
        "os":      true,
        "web":     true,
        "vuln":    true,
    }

    return validTypes[stageType]
}
```

### 6.5 TemplateGenerator 实现

```go
package pipeline

import (
    "fmt"
    "os"
)

// templateGenerator 模板样板生成器
type templateGenerator struct{}

// NewTemplateGenerator 创建模板样板生成器
func NewTemplateGenerator() TemplateGenerator {
    return &templateGenerator{}
}

// GenerateTemplate 生成模板样板文件
func (g *templateGenerator) GenerateTemplate(path string) error {
    content := g.GenerateTemplateContent()

    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
        return fmt.Errorf("failed to write template file: %w", err)
    }

    return nil
}

// GenerateTemplateContent 生成模板样板内容
func (g *templateGenerator) GenerateTemplateContent() string {
    return `# Pipeline 自定义扫描模板
# 使用方法：neoAgent scan run -t <target> --template <this_file>

# 模板名称
name: "my_custom_scan"

# 模板描述
description: "我的自定义扫描流程"

# 超时时间（支持 m, h 单位）
timeout: 30m

# 扫描阶段列表
stages:
  # 阶段 1: 存活扫描
  - name: "Alive Scan"
    type: "alive"
    critical: true
    params:
      concurrency: 1000
      resolve_hostname: true

  # 阶段 2: 端口扫描
  - name: "Port Scan"
    type: "port"
    critical: true
    depends_on:
      - "Alive Scan"
    params:
      # 端口范围：1-65535, top1000, 80,443,8080 等
      port_range: "1-65535"
      # 扫描速率（每秒）
      rate: 1000
      # 是否在端口扫描时进行服务识别
      service_detect: false

  # 阶段 3: 服务识别
  - name: "Service Detect"
    type: "service"
    critical: false
    depends_on:
      - "Port Scan"
    # 跳过条件：没有开放端口时跳过
    skip_condition: "no_open_ports"
    params:
      # 服务识别速率（每秒）
      rate: 100

  # 阶段 4: 操作系统识别
  - name: "OS Fingerprint"
    type: "os"
    critical: false
    depends_on:
      - "Service Detect"
    skip_condition: "no_open_ports"
    params:
      # 识别模式：auto, aggressive
      mode: "auto"

  # 阶段 5: Web 扫描
  - name: "Web Scan"
    type: "web"
    critical: false
    depends_on:
      - "Service Detect"
    # 跳过条件：没有 HTTP 服务时跳过
    skip_condition: "no_http_service"
    params:
      # 并发数
      concurrency: 10
      # 扫描深度
      depth: 3

  # 阶段 6: 漏洞扫描
  - name: "Vuln Scan"
    type: "vuln"
    critical: false
    depends_on:
      - "Port Scan"
    skip_condition: "no_open_ports"
    params:
      # 并发数
      concurrency: 5
      # 漏洞级别：low, medium, high, critical
      severity: "high,critical"

# ====== 字段说明 ======
#
# name: 阶段名称（唯一标识）
# type: 阶段类型（alive/port/service/os/web/vuln）
# critical: 是否为关键阶段（关键阶段失败则终止扫描）
# depends_on: 依赖的前置阶段名称列表
# skip_condition: 跳过条件（见下方支持的条件表达式）
# params: 阶段参数（根据阶段类型不同而不同）
#
# ====== 支持的跳过条件 ======
#
# no_open_ports: 没有开放端口
# no_http_service: 没有 HTTP 服务
# no_ssh_service: 没有 SSH 服务
# no_db_service: 没有数据库服务
# alive_failed: 存活扫描失败
#
# ====== 支持的阶段类型 ======
#
# alive: 存活扫描
#   - concurrency: 并发数
#   - resolve_hostname: 是否解析主机名
#
# port: 端口扫描
#   - port_range: 端口范围（1-65535, top1000, 80,443,8080 等）
#   - rate: 扫描速率（每秒）
#   - service_detect: 是否进行服务识别
#
# service: 服务识别
#   - rate: 识别速率（每秒）
#
# os: 操作系统识别
#   - mode: 识别模式（auto, aggressive）
#
# web: Web 扫描
#   - concurrency: 并发数
#   - depth: 扫描深度
#
# vuln: 漏洞扫描
#   - concurrency: 并发数
#   - severity: 漏洞级别（low, medium, high, critical）
#
`
}
```

### 6.6 AutoRunner 实现

```go
package pipeline

import (
    "context"
    "fmt"
    "sync"

    "neoagent/internal/pkg/logger"
)

// autoRunner 自动化执行器
type autoRunner struct {
    targetGenerator <-chan string
    concurrency     int
    presetManager   PresetManager
    showSummary     bool

    summaryMu sync.Mutex
    summaries []*PipelineContext
}

// NewAutoRunner 创建 AutoRunner
func NewAutoRunner(targetInput string, concurrency int, presetManager PresetManager, showSummary bool) AutoRunner {
    return &autoRunner{
        targetGenerator: GenerateTargets(targetInput),
        concurrency:     concurrency,
        presetManager:   presetManager,
        showSummary:     showSummary,
        summaries:       make([]*PipelineContext, 0),
    }
}

// RunWithPreset 使用预设模板执行扫描
func (r *autoRunner) RunWithPreset(ctx context.Context, presetName string, target string) ([]*PipelineContext, error) {
    // 获取预设模板
    preset, err := r.presetManager.GetBuiltinPreset(presetName)
    if err != nil {
        return nil, fmt.Errorf("failed to get preset: %w", err)
    }

    return r.RunWithConfig(ctx, preset, target)
}

// RunWithConfig 使用自定义配置执行扫描
func (r *autoRunner) RunWithConfig(ctx context.Context, config *Preset, target string) ([]*PipelineContext, error) {
    // 验证配置
    if err := r.presetManager.ValidatePreset(config); err != nil {
        return nil, fmt.Errorf("invalid preset config: %w", err)
    }

    // 创建阶段实例
    stages, err := r.createStages(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create stages: %w", err)
    }

    // 执行扫描
    var wg sync.WaitGroup
    sem := make(chan struct{}, r.concurrency)

    for ip := range r.targetGenerator {
        wg.Add(1)
        sem <- struct{}{}

        go func(targetIP string) {
            defer wg.Done()
            defer func() { <-sem }()

            // 创建 Pipeline Context
            pCtx := NewPipelineContext(targetIP)

            // 执行流水线
            r.executePipeline(ctx, pCtx, stages)

            // 收集结果
            if r.showSummary {
                r.summaryMu.Lock()
                r.summaries = append(r.summaries, pCtx)
                r.summaryMu.Unlock()
            }
        }(ip)
    }

    wg.Wait()

    // 输出最终总结报告
    if r.showSummary {
        r.printFinalReport()
    }

    return r.summaries, nil
}

// createStages 根据配置创建阶段实例
func (r *autoRunner) createStages(config *Preset) ([]PipelineStage, error) {
    var stages []PipelineStage

    for _, stageConfig := range config.Stages {
        var stage PipelineStage
        var err error

        switch stageConfig.Type {
        case "alive":
            stage, err = NewAliveStage(stageConfig)
        case "port":
            stage, err = NewPortStage(stageConfig)
        case "service":
            stage, err = NewServiceStage(stageConfig)
        case "os":
            stage, err = NewOSStage(stageConfig)
        case "web":
            stage, err = NewWebStage(stageConfig)
        case "vuln":
            stage, err = NewVulnStage(stageConfig)
        default:
            return nil, fmt.Errorf("unsupported stage type: %s", stageConfig.Type)
        }

        if err != nil {
            return nil, fmt.Errorf("failed to create stage %s: %w", stageConfig.Name, err)
        }

        stages = append(stages, stage)
    }

    return stages, nil
}

// executePipeline 执行单个 IP 的流水线逻辑
func (r *autoRunner) executePipeline(ctx context.Context, pCtx *PipelineContext, stages []PipelineStage) {
    // 按依赖顺序执行阶段
    for _, stage := range stages {
        logger.Debugf("[%s] Checking stage: %s", pCtx.IP, stage.Name())

        // 检查是否跳过
        if stage.ShouldSkip(pCtx) {
            logger.Debugf("[%s] Skipping stage: %s (condition not met)", pCtx.IP, stage.Name())
            continue
        }

        // 执行阶段
        logger.Infof("[%s] Executing stage: %s", pCtx.IP, stage.Name())
        if err := stage.Execute(ctx, pCtx); err != nil {
            logger.Errorf("[%s] Stage %s failed: %v", pCtx.IP, stage.Name(), err)

            // 关键阶段失败则终止
            if stage.IsCritical() {
                logger.Errorf("[%s] Critical stage failed, terminating pipeline", pCtx.IP)
                return
            }
        }
    }
}

// printFinalReport 输出最终总结报告
func (r *autoRunner) printFinalReport() {
    r.summaryMu.Lock()
    defer r.summaryMu.Unlock()

    fmt.Println("\n========== Pipeline Summary ==========")
    fmt.Printf("Total Targets: %d\n", len(r.summaries))

    aliveCount := 0
    totalPorts := 0
    totalServices := 0
    totalVulns := 0

    for _, ctx := range r.summaries {
        if ctx.Alive {
            aliveCount++
            totalPorts += len(ctx.OpenPorts)
            totalServices += len(ctx.Services)
            totalVulns += len(ctx.VulnResults)
        }
    }

    fmt.Printf("Alive Hosts: %d\n", aliveCount)
    fmt.Printf("Total Open Ports: %d\n", totalPorts)
    fmt.Printf("Total Services: %d\n", totalServices)
    fmt.Printf("Total Vulnerabilities: %d\n", totalVulns)
    fmt.Println("=======================================")
}
```

---

## 七、CLI 集成示例

### 7.1 命令注册

```go
package cmd

import (
    "github.com/spf13/cobra"
    "neoagent/internal/core/pipeline"
)

var (
    presetName   string
    templatePath string
)

// scanCmd 扫描命令
var scanCmd = &cobra.Command{
    Use:   "scan",
    Short: "执行扫描任务",
}

// scanRunCmd 执行扫描命令
var scanRunCmd = &cobra.Command{
    Use:   "run -t <target> [--preset <name> | --template <path>]",
    Short: "执行扫描任务",
    Args:  cobra.ExactArgs(0),
    RunE: func(cmd *cobra.Command, args []string) error {
        // 创建 PresetManager
        presetManager := pipeline.NewPresetManager()

        // 获取扫描配置
        var preset *pipeline.Preset
        var err error

        if presetName != "" {
            preset, err = presetManager.GetBuiltinPreset(presetName)
            if err != nil {
                return err
            }
        } else if templatePath != "" {
            preset, err = presetManager.LoadExternalTemplate(templatePath)
            if err != nil {
                return err
            }
        } else {
            return fmt.Errorf("must specify either --preset or --template")
        }

        // 创建 AutoRunner 并执行扫描
        runner := pipeline.NewAutoRunner(target, concurrency, presetManager, true)
        _, err = runner.RunWithConfig(context.Background(), preset, target)
        return err
    },
}

// scanTemplateListCmd 列出内置模板命令
var scanTemplateListCmd = &cobra.Command{
    Use:   "list",
    Short: "列出所有内置预设模板",
    Args:  cobra.ExactArgs(0),
    RunE: func(cmd *cobra.Command, args []string) error {
        presetManager := pipeline.NewPresetManager()
        presets := presetManager.ListBuiltinPresets()

        fmt.Println("Available Presets:")
        fmt.Println("==================")
        for _, preset := range presets {
            fmt.Printf("  - %s: %s (%d stages, %v)\n",
                preset.Name, preset.Description, preset.StageCount, preset.Timeout)
        }

        return nil
    },
}

// scanTemplateShowCmd 查看内置模板详情命令
var scanTemplateShowCmd = &cobra.Command{
    Use:   "show <preset_name>",
    Short: "查看内置预设模板详情",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        presetName := args[0]
        presetManager := pipeline.NewPresetManager()
        preset, err := presetManager.GetBuiltinPreset(presetName)
        if err != nil {
            return err
        }

        fmt.Printf("Preset: %s\n", preset.Name)
        fmt.Printf("Description: %s\n", preset.Description)
        fmt.Printf("Timeout: %v\n", preset.Timeout)
        fmt.Println("\nStages:")
        for i, stage := range preset.Stages {
            fmt.Printf("  %d. %s (%s)\n", i+1, stage.Name, stage.Type)
            fmt.Printf("     Critical: %v\n", stage.Critical)
            if len(stage.DependsOn) > 0 {
                fmt.Printf("     Depends On: %v\n", stage.DependsOn)
            }
            if stage.SkipCondition != "" {
                fmt.Printf("     Skip Condition: %s\n", stage.SkipCondition)
            }
        }

        return nil
    },
}

// scanTemplateCreateCmd 生成模板样板命令
var scanTemplateCreateCmd = &cobra.Command{
    Use:   "create <output_path>",
    Short: "生成模板样板文件",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        outputPath := args[0]
        generator := pipeline.NewTemplateGenerator()
        return generator.GenerateTemplate(outputPath)
    },
}

// scanTemplateCmd 模板管理命令
var scanTemplateCmd = &cobra.Command{
    Use:   "template",
    Short: "模板管理",
}

func init() {
    // scan run 命令参数
    scanRunCmd.Flags().StringVarP(&target, "target", "t", "", "扫描目标（IP、CIDR、域名）")
    scanRunCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 100, "并发数")
    scanRunCmd.Flags().StringVar(&presetName, "preset", "", "使用内置预设模板")
    scanRunCmd.Flags().StringVar(&templatePath, "template", "", "使用外部 YAML 模板")
    scanRunCmd.MarkFlagsMutuallyExclusive("preset", "template")

    // 注册子命令
    scanTemplateCmd.AddCommand(scanTemplateListCmd)
    scanTemplateCmd.AddCommand(scanTemplateShowCmd)
    scanTemplateCmd.AddCommand(scanTemplateCreateCmd)

    scanCmd.AddCommand(scanRunCmd)
    scanCmd.AddCommand(scanTemplateCmd)

    rootCmd.AddCommand(scanCmd)
}
```

---

## 八、实施计划

### 8.1 总体时间规划

**总计约 14 天**，分为 3 个阶段：

#### 第一阶段：核心基础（5 天）

**Day 1-2: PipelineContext 和 PipelineStage**
- [ ] 实现 PipelineContext 数据结构
- [ ] 实现 PipelineStage 接口
- [ ] 实现条件判断逻辑

**Day 3-4: PresetManager 和内置模板**
- [ ] 实现 PresetManager 接口
- [ ] 实现内置预设模板（hardcode）
- [ ] 实现模板验证逻辑

**Day 5: ExternalTemplateLoader**
- [ ] 实现 ExternalTemplateLoader 接口
- [ ] 实现 YAML 解析
- [ ] 实现模板格式验证

#### 第二阶段：核心功能（5 天）

**Day 6-7: TemplateGenerator**
- [ ] 实现 TemplateGenerator 接口
- [ ] 实现模板样板生成
- [ ] 编写详细的模板说明文档

**Day 8-9: AutoRunner**
- [ ] 实现 AutoRunner 接口
- [ ] 实现流水线执行逻辑
- [ ] 实现条件跳过逻辑

**Day 10: 各个 Stage 的基础实现**
- [ ] 实现 AliveStage
- [ ] 实现 PortStage
- [ ] 实现 ServiceStage
- [ ] 实现 OSStage
- [ ] 实现 WebStage
- [ ] 实现 VulnStage

#### 第三阶段：集成完善（4 天）

**Day 11-12: CLI 命令集成**
- [ ] 实现 scan run 命令
- [ ] 实现 scan template list 命令
- [ ] 实现 scan template show 命令
- [ ] 实现 scan template create 命令
- [ ] 实现命令参数验证

**Day 13-14: 测试和文档**
- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] 编写用户文档
- [ ] 性能测试和优化

### 8.2 验收标准

#### 功能验收

- [ ] 支持使用内置预设模板执行扫描
- [ ] 支持使用外部 YAML 模板执行扫描
- [ ] 支持生成模板样板文件
- [ ] 支持列出所有内置预设模板
- [ ] 支持查看内置预设模板详情
- [ ] 条件跳过功能正常工作
- [ ] 关键阶段失败能够终止扫描

#### 质量验收

- [ ] 单元测试覆盖率 > 80%
- [ ] 所有 CLI 命令正常工作
- [ ] 文档完整准确
- [ ] 代码符合项目规范
- [ ] 性能满足要求

---

## 九、使用示例

### 9.1 基本使用

```bash
# 使用内置全量扫描模板
./neoAgent scan run -t 192.168.1.0/24 --preset full_scan -c 100

# 使用内置快速扫描模板
./neoAgent scan run -t 192.168.1.0/24 --preset quick_scan -c 500

# 使用内置 Web 专项扫描模板
./neoAgent scan run -t 192.168.1.1 --preset web_scan -c 50

# 使用内置漏洞专项扫描模板
./neoAgent scan run -t 192.168.1.0/24 --preset vuln_scan -c 50
```

### 9.2 外部模板使用

```bash
# 生成模板样板
./neoAgent scan template create ./my_template.yaml

# 编辑模板样板
vim ./my_template.yaml

# 使用自定义模板执行扫描
./neoAgent scan run -t 192.168.1.0/24 --template ./my_template.yaml -c 100
```

### 9.3 模板管理

```bash
# 列出所有内置模板
./neoAgent scan template list

# 查看内置模板详情
./neoAgent scan template show full_scan
./neoAgent scan template show quick_scan
```

---

## 十、总结

本规划方案的核心优势：

1. **零配置文件依赖**：内置模板硬编码到代码中，二进制文件独立运行
2. **灵活扩展**：支持外部 YAML 模板，满足自定义需求
3. **易于使用**：提供模板样板生成功能，降低学习成本
4. **简单实用**：线性流程 + 条件跳过，避免过度设计
5. **完整功能**：支持常用扫描流程和自定义扩展

这个方案既保留了原版方案的预设模板思想，又避免了过度设计，非常适合 Agent 端的使用场景！
