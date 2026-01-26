package nmap

import (
	"strconv"
	"strings"
)

// OSMatchResult 匹配结果
type OSMatchResult struct {
	Fingerprint *OSFingerprint
	Accuracy    float64 // 0-100
}

// Match 在数据库中查找最佳匹配的指纹
func (db *OSDB) Match(target *OSFingerprint) *OSMatchResult {
	var bestMatch *OSFingerprint
	var bestScore float64 = -1.0

	// 遍历数据库中的每个指纹规则
	for _, ruleFP := range db.Fingerprints {
		score := calculateScore(target, ruleFP)
		if score > bestScore {
			bestScore = score
			bestMatch = ruleFP
		}
	}

	if bestMatch != nil {
		return &OSMatchResult{
			Fingerprint: bestMatch,
			Accuracy:    bestScore,
		}
	}

	return nil
}

// calculateScore 计算目标指纹与规则指纹的匹配度 (0-100)
// target: 扫描得到的指纹 (包含具体值)
// rule: 数据库中的指纹规则 (包含匹配模式)
func calculateScore(target, rule *OSFingerprint) float64 {
	totalTests := 0
	matchedTests := 0

	// 需要比较的测试项列表
	// Nmap 包含: SEQ, OPS, WIN, T1-T7, ECN, IE, U1
	testNames := []string{"SEQ", "OPS", "WIN", "T1", "T2", "T3", "T4", "T5", "T6", "T7", "ECN", "IE", "U1"}

	for _, name := range testNames {
		targetBody, hasTarget := target.MatchRule[name]
		ruleBody, hasRule := rule.MatchRule[name]

		// 如果双方都没有该测试项，不算分也不扣分
		if !hasTarget && !hasRule {
			continue
		}

		totalTests++

		// 一方有一方没有 -> 不匹配
		if hasTarget != hasRule {
			continue
		}

		// 双方都有，比较细节
		if matchTest(targetBody, ruleBody) {
			matchedTests++
		}
	}

	if totalTests == 0 {
		return 0
	}

	return (float64(matchedTests) / float64(totalTests)) * 100.0
}

// matchTest 比较单个测试项 (e.g. T1(R=Y%DF=N...))
func matchTest(targetBody, ruleBody string) bool {
	// 解析为 Map
	targetMap := ParseRuleBody(targetBody)
	ruleMap := ParseRuleBody(ruleBody)

	// 遍历规则中的所有属性
	for key, rulePattern := range ruleMap {
		targetVal, ok := targetMap[key]
		if !ok {
			// 规则要求有该属性，但目标没有 -> 不匹配
			return false
		}

		if !matchValue(targetVal, rulePattern) {
			return false
		}
	}

	return true
}

// matchValue 比较属性值
// targetVal: 具体值 (e.g. "100", "Y", "F")
// rulePattern: 模式 (e.g. "100-200", "Y|N", ">10")
func matchValue(targetVal, rulePattern string) bool {
	// 1. 处理逻辑或 (|)
	if strings.Contains(rulePattern, "|") {
		options := strings.Split(rulePattern, "|")
		for _, opt := range options {
			if matchValue(targetVal, opt) {
				return true
			}
		}
		return false
	}

	// 2. 处理范围 (-)
	if strings.Contains(rulePattern, "-") {
		// Nmap OS DB 使用十六进制数值
		parts := strings.SplitN(rulePattern, "-", 2)
		min, err1 := parseHexInt(parts[0])
		max, err2 := parseHexInt(parts[1])
		val, err3 := parseHexInt(targetVal)

		if err1 == nil && err2 == nil && err3 == nil {
			return val >= min && val <= max
		}
	}

	// 3. 处理大于 (>)
	if strings.HasPrefix(rulePattern, ">") {
		limit, err := parseHexInt(rulePattern[1:])
		val, err2 := parseHexInt(targetVal)
		if err == nil && err2 == nil {
			return val > limit
		}
	}

	// 4. 处理小于 (<)
	if strings.HasPrefix(rulePattern, "<") {
		limit, err := parseHexInt(rulePattern[1:])
		val, err2 := parseHexInt(targetVal)
		if err == nil && err2 == nil {
			return val < limit
		}
	}

	// 5. 精确匹配
	return targetVal == rulePattern
}

func parseHexInt(s string) (int, error) {
	v, err := strconv.ParseInt(s, 16, 64)
	return int(v), err
}
