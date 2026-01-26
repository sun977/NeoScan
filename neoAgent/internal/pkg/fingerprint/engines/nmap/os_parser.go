package nmap

import (
	"bufio"
	"strings"
)

// OSFingerprint 代表 Nmap nmap-os-db 中的一条指纹记录
type OSFingerprint struct {
	Name      string            // Fingerprint name (e.g., "Linux 2.6.x")
	Class     string            // Class line (Vendor | OS Family | OS Gen | Device Type)
	CPE       string            // CPE line
	MatchRule map[string]string // Key: TestName (SEQ, T1...), Value: TestRule string
}

// OSDB 存储解析后的 OS 指纹库
type OSDB struct {
	Fingerprints []*OSFingerprint
	MatchPoints  map[string]int // 权重配置 (MatchPoints 指令)
}

// ParseOSDB 解析 nmap-os-db 内容
func ParseOSDB(content string) (*OSDB, error) {
	db := &OSDB{
		Fingerprints: make([]*OSFingerprint, 0),
		MatchPoints:  make(map[string]int),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentFP *OSFingerprint

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 处理 MatchPoints 指令
		if strings.HasPrefix(line, "MatchPoints") {
			// TODO: 解析权重配置，目前暂时跳过，使用默认权重
			continue
		}

		// 处理 Fingerprint 指令 (新指纹开始)
		if strings.HasPrefix(line, "Fingerprint ") {
			if currentFP != nil {
				db.Fingerprints = append(db.Fingerprints, currentFP)
			}
			currentFP = &OSFingerprint{
				Name:      strings.TrimPrefix(line, "Fingerprint "),
				MatchRule: make(map[string]string),
			}
			continue
		}

		// 如果没有当前指纹上下文，跳过后续指令
		if currentFP == nil {
			continue
		}

		// 处理 Class 指令
		if strings.HasPrefix(line, "Class ") {
			currentFP.Class = strings.TrimPrefix(line, "Class ")
			continue
		}

		// 处理 CPE 指令
		if strings.HasPrefix(line, "CPE ") {
			currentFP.CPE = strings.TrimPrefix(line, "CPE ")
			continue
		}

		// 处理测试规则 (SEQ, OPS, WIN, T1, etc.)
		// 格式: TestName(Rule)
		// e.g. SEQ(SP=25%GCD=75...)
		if idx := strings.Index(line, "("); idx > 0 && strings.HasSuffix(line, ")") {
			testName := line[:idx]
			ruleBody := line[idx+1 : len(line)-1]
			currentFP.MatchRule[testName] = ruleBody
		}
	}

	// 添加最后一个指纹
	if currentFP != nil {
		db.Fingerprints = append(db.Fingerprints, currentFP)
	}

	return db, scanner.Err()
}

// ParseRuleBody 解析规则体字符串 (e.g., "R=Y%DF=N%T=FA-104")
// 返回 Key-Value 映射
func ParseRuleBody(body string) map[string]string {
	rules := make(map[string]string)
	parts := strings.Split(body, "%")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			rules[kv[0]] = kv[1]
		}
	}
	return rules
}
