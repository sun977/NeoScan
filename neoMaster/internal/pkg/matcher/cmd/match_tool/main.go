// 测试方式
// go run internal/pkg/matcher/cmd/match_tool/main.go -rule rule_data.json -data data_match.json -v
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"neomaster/internal/pkg/matcher"
)

func main() {
	rulePath := flag.String("rule", "", "Path to rule JSON file or directory")
	dataPath := flag.String("data", "", "Path to data JSON file")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	if *rulePath == "" || *dataPath == "" {
		fmt.Println("Usage: match_tool -rule <rule_file_or_dir> -data <data_file> [-v]")
		os.Exit(1)
	}

	// 1. Load Data
	data, err := loadData(*dataPath)
	if err != nil {
		fmt.Printf("❌ Failed to load data: %v\n", err)
		os.Exit(1)
	}
	if *verbose {
		fmt.Printf("✅ Data loaded from %s\n", *dataPath)
	}

	// 2. Process Rules
	info, err := os.Stat(*rulePath)
	if err != nil {
		fmt.Printf("❌ Error checking rule path: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		processDir(*rulePath, data, *verbose)
	} else {
		processFile(*rulePath, data, *verbose)
	}
}

func loadData(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func processDir(dirPath string, data map[string]interface{}, verbose bool) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("❌ Failed to read directory: %v\n", err)
		return
	}

	total := 0
	matchedCount := 0
	errorCount := 0

	fmt.Printf("📂 Processing rules in directory: %s\n", dirPath)
	fmt.Println("---------------------------------------------------")

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		total++
		filePath := filepath.Join(dirPath, f.Name())
		result, err := runMatch(filePath, data)

		status := "❌ ERROR"
		if err == nil {
			if result {
				status = "✅ MATCH"
				matchedCount++
			} else {
				status = "⚪ SKIP "
			}
		} else {
			errorCount++
		}

		fmt.Printf("[%s] %s\n", status, f.Name())
		if err != nil && verbose {
			fmt.Printf("   Error: %v\n", err)
		}
	}

	fmt.Println("---------------------------------------------------")
	fmt.Printf("Summary: Total=%d, Matched=%d, Errors=%d\n", total, matchedCount, errorCount)
}

func processFile(filePath string, data map[string]interface{}, verbose bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("❌ Error reading file %s: %v\n", filePath, err)
		os.Exit(2)
	}

	// 1. Try to parse as array of rules
	var rules []matcher.MatchRule
	if err := json.Unmarshal(content, &rules); err == nil {
		// Verify it is actually a JSON array (Unmarshal might succeed for null)
		// Simple check: trim space and check first char
		trimmed := strings.TrimSpace(string(content))
		if strings.HasPrefix(trimmed, "[") {
			processRulesArray(filePath, rules, data, verbose)
			return
		}
	}

	// 2. Try to parse as single rule
	rule, err := matcher.ParseJSON(string(content))
	if err != nil {
		fmt.Printf("❌ Error parsing rule %s: %v\n", filePath, err)
		os.Exit(2)
	}

	matched, err := matcher.Match(data, rule)
	if err != nil {
		fmt.Printf("❌ Error processing rule %s: %v\n", filePath, err)
		os.Exit(2)
	}

	if matched {
		fmt.Printf("✅ MATCH: %s matches the data.\n", filePath)
		os.Exit(0)
	} else {
		fmt.Printf("⚪ NO MATCH: %s does not match the data.\n", filePath)
		os.Exit(1)
	}
}

func processRulesArray(filePath string, rules []matcher.MatchRule, data map[string]interface{}, verbose bool) {
	fmt.Printf("📂 Processing rule file: %s (containing %d rules)\n", filePath, len(rules))
	matchedCount := 0
	errorCount := 0

	for i, rule := range rules {
		matched, err := matcher.Match(data, rule)
		if err != nil {
			fmt.Printf("   [Rule %d] ❌ ERROR: %v\n", i, err)
			errorCount++
		} else if matched {
			if verbose {
				fmt.Printf("   [Rule %d] ✅ MATCH\n", i)
			}
			matchedCount++
		} else {
			if verbose {
				fmt.Printf("   [Rule %d] ⚪ NO MATCH\n", i)
			}
		}
	}

	fmt.Printf("Summary: %d/%d matched, %d errors\n", matchedCount, len(rules), errorCount)
	if errorCount > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

func runMatch(ruleFile string, data map[string]interface{}) (bool, error) {
	content, err := os.ReadFile(ruleFile)
	if err != nil {
		return false, fmt.Errorf("read file error: %v", err)
	}

	rule, err := matcher.ParseJSON(string(content))
	if err != nil {
		return false, fmt.Errorf("parse json error: %v", err)
	}

	return matcher.Match(data, rule)
}
