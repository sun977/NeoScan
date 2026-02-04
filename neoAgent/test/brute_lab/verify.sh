#!/bin/bash
# NeoAgent Brute Force Verification Script (Linux/Bash)
# 用于验证 docker-compose.yml 搭建的靶场环境

set -e # 遇到错误立即退出

AGENT_PATH="./neoAgent"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 编译 Agent
echo -e "${CYAN}[*] Compiling neoAgent...${NC}"
go build -o neoAgent ./cmd/agent
if [ $? -ne 0 ]; then
    echo -e "${RED}Compilation failed!${NC}"
    exit 1
fi
echo -e "${GREEN}[+] Compilation success.${NC}"

# 2. 定义测试目标
# 格式: Service Port User Pass
TARGETS=(
    "ssh          2222   testuser   password123"
    "mysql        33060  root       password123"
    "redis        63790  NONE       password123" # NONE 表示空用户
    "postgres     54320  postgres   password123"
    "ftp          2121   testuser   password123"
    "mongo        27017  admin      password123"
    "clickhouse   9000   testuser   password123"
    "smb          4455   testuser   password123"
    "elasticsearch 9200  elastic    password123"
    "snmp         1610   NONE       public"      # NONE 表示空用户
)

# 3. 执行测试
echo -e "\n${CYAN}[*] Starting Brute Force Tests against Localhost (Docker Lab)...${NC}"
echo -e "${YELLOW}Ensure you have started the lab with: docker-compose up -d${NC}\n"

# 循环遍历测试目标
for target in "${TARGETS[@]}"; do
    # 读取参数
    read -r svc port user pass <<< "$target"

    cmd_args=("scan" "brute" "-t" "127.0.0.1" "-p" "$port" "-s" "$svc" "--stop-on-success")

    if [ "$user" != "NONE" ]; then
        cmd_args+=("--users" "$user")
    fi
    if [ "$pass" != "NONE" ]; then
        cmd_args+=("--pass" "$pass")
    fi

    echo -n "[-] Testing $svc on port $port... "

    # 执行命令并捕获输出
    # 2>&1 将 stderr 重定向到 stdout
    output=$($AGENT_PATH "${cmd_args[@]}" 2>&1)

    # 检查结果
    if echo "$output" | grep -q "FOUND"; then
        echo -e "${GREEN}[PASS]${NC}"
    else
        echo -e "${RED}[FAIL]${NC}"
        echo "    Command: $AGENT_PATH ${cmd_args[*]}"
        echo "    Output snippet:"
        echo "$output" | head -n 5 | sed 's/^/        /'
    fi
done

echo -e "\n${CYAN}[*] Test Complete.${NC}"
