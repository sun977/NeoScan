# NeoAgent Brute Force Integration Lab

这是一个基于 Docker Compose 的集成测试靶场，用于验证 NeoAgent 对各种协议弱口令爆破的有效性。

## 前置条件
*   已安装 Docker Desktop (Windows) 或 Docker Engine (Linux/Mac)。
*   已安装 PowerShell (Windows 自带) 或 pwsh。

## 快速开始

### 1. 启动靶场
在 `test/brute_lab` 目录下运行：
```bash
docker-compose up -d
```
这将启动约 10 个容器，映射到本地的高端口（如 SSH 映射到 2222，MySQL 映射到 33060），避免与本机服务冲突。

### 2. 运行验证脚本
回到项目根目录 `neoAgent/`，运行：
```powershell
.\test\brute_lab\verify.ps1
```
脚本会自动：
1.  编译最新的 `neoAgent.exe`。
2.  针对每个服务执行爆破测试。
3.  输出 PASS/FAIL 结果。

### 3. 清理环境
测试完成后，销毁靶场以释放资源：
```bash
docker-compose down
```

## 包含的服务与凭据

| 服务 | 端口 (Local) | 用户名 | 密码 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| SSH | 2222 | testuser | password123 | |
| MySQL | 33060 | root | password123 | |
| Redis | 63790 | (无) | password123 | |
| Postgres | 54320 | postgres | password123 | |
| FTP | 2121 | testuser | password123 | |
| MongoDB | 27017 | admin | password123 | |
| ClickHouse| 9000 | testuser | password123 | |
| SMB | 4455 | testuser | password123 | |
| Elasticsearch | 9200 | elastic | password123 | |
| SNMP | 1610 | (无) | public | UDP |
| RDP | 33890 | (默认) | (默认) | 仅验证协议握手 |

## 常见问题
*   **连接超时**: 首次启动容器可能需要拉取镜像，建议先手动执行 `docker-compose pull`。
*   **SMB 失败**: SMB 协议对端口转发比较敏感，且 Windows 本机可能会干扰 445 端口。如果 4455 失败，可能是容器配置问题。
