# NeoAgent Mock Lab Verification Script
# 用于在无 Docker 环境下验证 Agent 的基础爆破逻辑

$ErrorActionPreference = "Stop"
$AgentPath = ".\neoAgent.exe"
$MockServerPath = ".\mock_server.exe"

# 1. 编译 Agent
Write-Host "[*] Compiling neoAgent..." -ForegroundColor Cyan
go build -o neoAgent.exe ./cmd/agent
if ($LASTEXITCODE -ne 0) { Write-Error "Agent compilation failed!" }

# 2. 编译 Mock Server
Write-Host "[*] Compiling Mock Server..." -ForegroundColor Cyan
go build -o mock_server.exe ./test/mock_lab/main.go
if ($LASTEXITCODE -ne 0) { Write-Error "Mock Server compilation failed!" }

# 3. 启动 Mock Server
Write-Host "[*] Starting Mock Server..." -ForegroundColor Cyan
$MockProcess = Start-Process -FilePath $MockServerPath -PassThru -NoNewWindow
Start-Sleep -Seconds 2 # 等待端口监听

try {
    # 4. 定义测试目标
    # 注意：Mock Server 仅实现了 Redis 和 HTTP 的有效认证
    # SSH 和 MySQL 仅能测试连接性（预期是 Fail，但应该是协议错误而不是连接超时）
    $Targets = @(
        @{ Service="redis";        Port="63790"; User="";         Pass="password123"; Expected="FOUND" },
        @{ Service="elasticsearch";Port="9200";  User="elastic";  Pass="password123"; Expected="FOUND" },
        @{ Service="mysql";        Port="33061"; User="root";     Pass="password123"; Expected="FAIL" }, # Mock Server 返回垃圾数据
        @{ Service="rdp";          Port="33890"; User="root";     Pass="password123"; Expected="FAIL" }, # Mock Server 返回短数据 "Hello"
        @{ Service="ssh";          Port="2222";  User="root";     Pass="password123"; Expected="FAIL" }  # Mock Server 无法完成握手
    )

    foreach ($t in $Targets) {
        $svc = $t.Service
        $port = $t.Port
        $user = $t.User
        $pass = $t.Pass
        $expect = $t.Expected
        
        $cmdArgs = @("scan", "brute", "-t", "127.0.0.1", "-p", $port, "-s", $svc)
        if ($user -ne "") { $cmdArgs += ("--users", $user) }
        if ($pass -ne "") { $cmdArgs += ("--pass", $pass) }

        Write-Host "[-] Testing $svc on port $port (Expect: $expect)..." -NoNewline
        
        try {
            $output = & $AgentPath $cmdArgs 2>&1
        } catch {
            $output = $_.Exception.Message
        }

        $found = $output -match "FOUND"

        if (($expect -eq "FOUND" -and $found) -or ($expect -eq "FAIL" -and -not $found)) {
            Write-Host " [PASS]" -ForegroundColor Green
        } else {
            Write-Host " [FAIL]" -ForegroundColor Red
            Write-Host "    Output snippet: "
            $output | ForEach-Object { Write-Host "      $_" }
        }
    }

} finally {
    # 5. 清理
    Write-Host "`n[*] Stopping Mock Server..." -ForegroundColor Cyan
    Stop-Process -Id $MockProcess.Id -ErrorAction SilentlyContinue
    Remove-Item $MockServerPath -ErrorAction SilentlyContinue
}
