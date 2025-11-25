# 资产管理模块
### 3.4 资产管理模块

#### 功能描述
- 资产信息统一管理
- 资产发现和同步
- 资产清单维护
- 资产变更跟踪

#### 核心服务
- **资产同步服务** (`internal/service/asset/sync.go`)
  - 外部资产系统对接
  - 资产信息同步和更新
  - 数据一致性保证

- **资产发现服务** (`internal/service/asset/discovery.go`)
  - 网络资产自动发现
  - 服务和端口识别
  - 资产指纹识别

- **资产清单管理** (`internal/service/asset/inventory.go`)
  - 资产分类和标签
  - 资产生命周期管理
  - 资产关系图谱