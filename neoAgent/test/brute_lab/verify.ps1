# NeoAgent Brute Force Verification Script
# 用于验证 docker-compose.yml 搭建的靶场环境

$ErrorActionPreference = "Stop"
$AgentPath = ".\neoAgent.exe"

# 1. 编译 Agent
Write-Host "[*] Compiling neoAgent..." -ForegroundColor Cyan
go build -o neoAgent.exe ./cmd/agent
if ($LASTEXITCODE -ne 0) {
    Write-Error "Compilation failed!"
}
Write-Host "[+] Compilation success." -ForegroundColor Green

# 2. 定义测试目标
# 格式: @{ Service="ssh"; Port="2222"; User="testuser"; Pass="password123"; Expected="Success" }
$Targets = @(
    @{ Service="ssh";          Port="2222";  User="testuser"; Pass="password123" },
    @{ Service="mysql";        Port="33060"; User="root";     Pass="password123" },
    @{ Service="redis";        Port="63790"; User="";         Pass="password123" },
    @{ Service="postgres";     Port="54320"; User="postgres"; Pass="password123" },
    @{ Service="ftp";          Port="2121";  User="testuser"; Pass="password123" },
    @{ Service="mongo";        Port="27017"; User="admin";    Pass="password123" },
    @{ Service="clickhouse";   Port="9000";  User="testuser"; Pass="password123" },
    @{ Service="smb";          Port="4455";  User="testuser"; Pass="password123" },
    @{ Service="elasticsearch";Port="9200";  User="elastic";  Pass="password123" },
    # SNMP 和 RDP 可能需要特殊处理或等待更久
    @{ Service="snmp";         Port="1610";  User="";         Pass="public" } 
)

# 3. 执行测试
Write-Host "`n[*] Starting Brute Force Tests against Localhost (Docker Lab)..." -ForegroundColor Cyan
Write-Host "Ensure you have started the lab with: docker-compose up -d`n" -ForegroundColor Yellow

foreach ($t in $Targets) {
    $svc = $t.Service
    $port = $t.Port
    $user = $t.User
    $pass = $t.Pass
    
    $cmdArgs = @("scan", "brute", "-t", "127.0.0.1", "-p", $port, "-s", $svc, "--stop-on-success")
    
    if ($user -ne "") {
        $cmdArgs += ("--users", $user)
    }
    if ($pass -ne "") {
        $cmdArgs += ("--pass", $pass)
    }

    Write-Host "[-] Testing $svc on port $port..." -NoNewline
    
    # 执行命令并捕获输出
    $output = & $AgentPath $cmdArgs 2>&1
    
    # 检查结果
    if ($output -match "FOUND") {
        Write-Host " [PASS]" -ForegroundColor Green
    } else {
        Write-Host " [FAIL]" -ForegroundColor Red
        Write-Host "    Command: $AgentPath $cmdArgs"
        Write-Host "    Output snippet: $($output | Select-Object -First 5)"
    }
}

Write-Host "`n[*] Test Complete."
