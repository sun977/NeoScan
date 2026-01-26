# fscan 端口扫描实现方式和工作原理分析

## 概述

fscan 的端口扫描功能是其核心功能之一，通过高效的并发机制实现对目标主机端口的快速探测。该功能位于 [Plugins/portscan.go](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/fscan/Plugins/portscan.go) 文件中。

## 核心数据结构

```go
type Addr struct {
    ip   string
    port int
}
```

- 用于封装IP地址和端口信息
- 作为任务单元在并发系统中传输

## 主要函数分析

### 1. PortScan 函数

```go
func PortScan(hostslist []string, ports string, timeout int64) []string
```

这是端口扫描的主函数，主要功能包括：

- 解析端口字符串
- 过滤不需要扫描的端口
- 创建并发任务管道
- 启动多个 worker 协程进行扫描
- 收集扫描结果

### 2. PortConnect 函数

```go
func PortConnect(addr Addr, respondingHosts chan<- string, adjustedTimeout int64, wg *sync.WaitGroup)
```

负责对单个 IP:Port 进行连接测试，尝试建立 TCP 连接。

### 3. NoPortScan 函数

```go
func NoPortScan(hostslist []string, ports string) (AliveAddress []string)
```

不实际扫描端口，只是将 IP:Port 组合直接返回，通常用于 webonly 或 webpoc 模式。

## 端口扫描工作流程

```
                    +------------------+
                    |  输入主机列表     |
                    |  hostslist       |
                    +--------+---------+
                             |
                             v
                    +------------------+
                    |  解析端口字符串    |
                    |  ports string     |
                    +--------+---------+
                             |
                             v
                    +------------------+
                    |  过滤排除端口     |
                    |  noPorts        |
                    +--------+---------+
                             |
                             v
                    +------------------+
                    |  构建任务队列     |
                    |  Addrs chan     |
                    +--------+---------+
                             |
            +---------------+---------------+
            |                               |
            v                               v
    +---------------+               +---------------+
    |  Worker 1    |               |  Worker N     |
    |  扫描任务      |               |  扫描任务      |
    +-------+-------+               +-------+-------+
            |                               |
            +---------------+---------------+
                            |
                            v
                    +------------------+
                    |  结果收集        |
                    |  results chan   |
                    +--------+---------+
                             |
                             v
                    +------------------+
                    |  返回存活端口     |
                    |  AliveAddress   |
                    +------------------+
```

## 详细实现步骤

### 1. 端口解析与过滤

```go
probePorts := common.ParsePort(ports)  // 解析端口字符串为端口列表
noPorts := common.ParsePort(common.NoPorts)  // 获取要排除的端口列表
```

- 将端口字符串（如 "22,80,443" 或 "1-1000"）转换为整数端口列表
- 根据 noPorts 过滤掉不需要扫描的端口

### 2. 任务队列初始化

```go
workers := common.Threads  // 获取线程数，默认为 600
Addrs := make(chan Addr, len(hostslist)*len(probePorts))  // 任务队列
results := make(chan string, len(hostslist)*len(probePorts))  // 结果队列
```

- 创建两个 channel：Addrs 用于存放扫描任务，results 用于存放扫描结果
- channel 的缓冲区大小根据主机数量和端口数量确定

### 3. 结果收集协程

```go
go func() {
    for found := range results {
        AliveAddress = append(AliveAddress, found)
        wg.Done()
    }
}()
```

- 启动一个协程专门收集扫描结果
- 使用 WaitGroup 确保所有结果都被处理

### 4. 扫描工作协程

```go
for i := 0; i < workers; i++ {
    go func() {
        for addr := range Addrs {
            PortConnect(addr, results, timeout, &wg)
            wg.Done()
        }
    }()
}
```

- 根据线程数启动相应数量的扫描协程
- 每个协程不断从 Addrs channel 获取任务并执行扫描
- 扫描完成后将结果发送到 results channel

### 5. 任务分配

```go
for _, port := range probePorts {
    for _, host := range hostslist {
        wg.Add(1)
        Addrs <- Addr{host, port}
    }
}
```

- 将所有主机和端口的组合放入任务队列
- 每个任务增加 WaitGroup 计数

### 6. 端口连接测试

在 [PortConnect](file:///c%3A/Users/root/Desktop/code/GoCode/code_03/fscan/Plugins/portscan.go#L61-L73) 函数中:

```go
conn, err := common.WrapperTcpWithTimeout("tcp4", fmt.Sprintf("%s:%v", host, port), time.Duration(adjustedTimeout)*time.Second)
```

- 尝试建立 TCP 连接到指定的 IP:Port
- 使用设定的超时时间（默认3秒）
- 如果连接成功，记录为开放端口并发送到结果队列

## 并发控制机制

fscan 的端口扫描采用生产者-消费者模式：

1. **生产者**: 主协程将扫描任务（IP:Port）放入 Addrs channel
2. **消费者**: 多个工作协程从 Addrs channel 获取任务并执行扫描
3. **结果处理**: 扫描结果通过 results channel 返回

这种设计的优点：
- 有效控制并发数量，避免系统资源耗尽
- 通过 channel 实现任务的均衡分配
- 使用 WaitGroup 确保所有任务完成后再返回结果

## 性能优化

1. **并发控制**: 通过 Threads 参数控制最大并发数
2. **连接超时**: 设置合理的超时时间避免长时间挂起
3. **内存管理**: 通过定期 GC 释放内存
4. **任务队列**: 使用 channel 实现高效的生产者-消费者模式

## 总结

fscan 的端口扫描功能通过 Go 语言的并发特性实现了高效的端口探测，其设计合理、性能优异，能够快速准确地发现目标主机上的开放端口。