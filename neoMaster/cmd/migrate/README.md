# Database Migration Tool

## 简介 (Introduction)
这是 NeoMaster 项目的数据库迁移与数据填充工具。它负责同步 Go 代码中的模型定义到数据库表结构（Schema Migration），并根据环境初始化必要的基础数据（Data Seeding）。

> "Bad programmers worry about the code. Good programmers worry about data structures and their relationships." —— 这个工具的核心就是维护正确的数据结构。

## 功能 (Features)
- **Schema Migration**: 自动创建或更新数据库表结构（基于 GORM AutoMigrate）。
- **Data Seeding**: 初始化系统基础数据（如管理员账号、扫描类型配置、标签系统等）。
- **Environment Aware**: 支持多环境配置加载（test, dev, prod）。
- **Safety**: 危险操作（如 Drop Table）需显式开启。

## 快速开始 (Quick Start)

### 编译 (Build)
在 `neoMaster` 根目录下执行：
```bash
# 编译迁移工具
go build -o migrate.exe cmd/migrate/main.go
```

### 使用 (Usage)

**1. 初始化测试环境 (推荐开发使用)**
> ⚠️ **警告**: `-drop=true` 会删除所有现有表和数据！

这将清理旧数据，重建表结构，并填充测试数据。
```bash
# Windows
migrate.exe -env=test -drop=true -seed=true

# Linux/Mac
./migrate -env=test -drop=true -seed=true
```

**2. 生产环境更新**
仅更新表结构，不删除现有数据，不强制填充测试数据。
```bash
migrate.exe -env=prod -drop=false -seed=false
```

## 命令行参数 (Flags)

| 参数 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `-env` | string | `test` | 运行环境，决定加载哪个配置文件 (config_test.yaml 等) |
| `-drop` | bool | `false` | **[危险]** 是否在迁移前删除所有表结构 |
| `-seed` | bool | `true` | 是否在迁移后填充初始/测试数据 |
| `-verbose`| bool | `false` | 是否显示详细调试日志 |

## 初始化数据说明 (Seeded Data)

当开启 `-seed=true` 时，工具会使用 `FirstOrCreate` 策略初始化以下数据（避免重复）：

1.  **系统用户 (System User)**
    -   **管理员账号 (Admin)**: `admin`
        -   默认密码: `123456`
        -   用途: 人工管理系统，拥有最高权限。
    -   **系统内置账号 (System Internal)**: `sysuser`
        -   默认密码: `123456`
        -   用途: 系统内部自动化任务和服务间调用使用（非人工操作）。
    
2.  **Agent配置 (Agent Config)**
    -   **Scan Types**: 14种标准扫描类型 (如 `ipAliveScan`, `portScan`, `webBasicScan` 等)，同步自 SQL 定义。
        - 包含 `config_template` 默认配置。
        - 标记为 `is_system=true`。
    -   **Tag System**: 新版标签系统基础数据（如果已实现）。

3.  **编排引擎 (Orchestrator)**
    -   **ScanTool Templates**: 预置扫描工具模板 (Nmap, Masscan, Zmap 等)。
    -   **Projects**: 示例项目 "Default Project"。
    -   **Workflows**: 示例工作流 "Basic Network Scan"。

## 故障排查 (Troubleshooting)

- **"Table 'xxx' already exists"**: 通常 GORM 会自动处理，但如果表结构冲突严重，请尝试使用 `-drop=true` 重置（仅限开发环境）。
- **"Config load failed"**: 检查当前目录下或 `config/` 目录下是否存在对应环境的配置文件（如 `config_test.yaml`）。
- **"Database connection failed"**: 检查配置文件中的 MySQL 连接字符串、账号密码及网络连通性。

---
*遵循 "Never break userspace" 原则，所有迁移操作应保证向后兼容性（除非显式使用 drop 模式）。*
