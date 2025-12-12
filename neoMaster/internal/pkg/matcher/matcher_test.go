package matcher

import (
	"encoding/json"
	"testing"
)

func TestMatch_UserSpecificRule(t *testing.T) {
	// 用户提供的特定规则
	jsonRule := `
 { 
   "and": [{ 
     "field": "device_type", 
     "operator": "equals", 
     "value": "honeypot" 
   }, { 
     "field": "os", 
     "operator": "contains", 
     "value": "linux" 
   }, { 
     "or": [{ 
       "field": "port_count", 
       "operator": "greater_than", 
       "value": "1000" 
     }, { 
       "field": "port_open", 
       "operator": "contains", 
       "value": "80" 
     }, { 
       "field": "service", 
       "operator": "contains", 
       "value": "sshd" 
     }, { 
       "field": "test_field", 
       "operator": "contains", 
       "value": "portmap" 
     }, { 
       "field": "test_field", 
       "operator": "regex", 
       "value": ".*(\\d+\\.){3}\\d+.*" 
     }] 
   }] 
 }
`
	rule, err := ParseJSON(jsonRule)
	if err != nil {
		t.Fatalf("Failed to parse JSON rule: %v", err)
	}

	// Case 1: 满足 AND条件 + OR中的 port_open (80)
	data1 := map[string]interface{}{
		"device_type": "honeypot",
		"os":          "ubuntu linux 20.04",
		"port_open":   []int{22, 80, 443},
		"port_count":  10,
	}
	matched, err := Match(data1, rule)
	if err != nil {
		t.Errorf("Case 1 Error: %v", err)
	}
	if !matched {
		t.Errorf("Case 1 Failed: Expected match via port_open=80")
	}

	// Case 2: 满足 AND条件 + OR中的 port_count > 1000
	// 注意: 规则中 value 是字符串 "1000"，我们已增强代码支持字符串数字比较
	data2 := map[string]interface{}{
		"device_type": "honeypot",
		"os":          "redhat linux",
		"port_count":  2000,
	}
	matched, err = Match(data2, rule)
	if err != nil {
		t.Errorf("Case 2 Error: %v", err)
	}
	if !matched {
		t.Errorf("Case 2 Failed: Expected match via port_count > 1000")
	}

	// Case 3: 满足 AND条件 + OR中的 regex (IP匹配)
	data3 := map[string]interface{}{
		"device_type": "honeypot",
		"os":          "linux",
		"test_field":  "Detected IP: 192.168.1.5 connected",
	}
	matched, err = Match(data3, rule)
	if err != nil {
		t.Errorf("Case 3 Error: %v", err)
	}
	if !matched {
		t.Errorf("Case 3 Failed: Expected match via regex IP")
	}

	// Case 4: 失败 - device_type 不匹配
	data4 := map[string]interface{}{
		"device_type": "firewall", // Mismatch
		"os":          "linux",
		"port_open":   []int{80},
	}
	matched, err = Match(data4, rule)
	if matched {
		t.Errorf("Case 4 Failed: Expected NO match due to device_type")
	}

	// Case 5: 失败 - os 不匹配
	data5 := map[string]interface{}{
		"device_type": "honeypot",
		"os":          "windows server", // Mismatch
		"port_open":   []int{80},
	}
	matched, err = Match(data5, rule)
	if matched {
		t.Errorf("Case 5 Failed: Expected NO match due to os")
	}

	// Case 6: 失败 - OR 条件全不满足
	data6 := map[string]interface{}{
		"device_type": "honeypot",
		"os":          "linux",
		"port_count":  500,           // < 1000
		"port_open":   []int{22, 23}, // No 80
		"service":     "ftp",         // No sshd
		"test_field":  "nothing",     // No portmap, no IP
	}
	matched, err = Match(data6, rule)
	if matched {
		t.Errorf("Case 6 Failed: Expected NO match due to all OR conditions failing")
	}
}

func TestMatch_ComplexNested(t *testing.T) {
	// 用户提供的复杂嵌套 JSON
	jsonRule := `
 { 
   "and": [{ 
     "field": "sourceProcessName", 
     "operator": "contains", 
     "value": "HaozipSvc.exe" 
   }, { 
     "and": [{ 
       "field": "destinationProcessName", 
       "operator": "equals", 
       "value": "C:\\Windows\\System32\\lsass.exe" 
     }] 
   }, { 
     "or": [{ 
       "field": "filePath", 
       "operator": "contains", 
       "value": "NT AUTHORITY\\SYSTEM" 
     }] 
   }] 
 }
`
	rule, err := ParseJSON(jsonRule)
	if err != nil {
		t.Fatalf("Failed to parse JSON rule: %v", err)
	}

	// Case 1: 完全匹配
	data1 := map[string]interface{}{
		"sourceProcessName":      "C:\\Program Files\\Haozip\\HaozipSvc.exe",
		"destinationProcessName": "C:\\Windows\\System32\\lsass.exe",
		"filePath":               "C:\\Users\\NT AUTHORITY\\SYSTEM\\test.txt",
	}
	matched, err := Match(data1, rule)
	if err != nil {
		t.Errorf("Match error: %v", err)
	}
	if !matched {
		t.Errorf("Expected match for data1, but got false")
	}

	// Case 2: sourceProcessName 不匹配 (第一个 AND 条件失败)
	data2 := map[string]interface{}{
		"sourceProcessName":      "notepad.exe",
		"destinationProcessName": "C:\\Windows\\System32\\lsass.exe",
		"filePath":               "C:\\Users\\NT AUTHORITY\\SYSTEM\\test.txt",
	}
	matched, err = Match(data2, rule)
	if !matched && err == nil {
		// Expected
	} else {
		t.Errorf("Expected no match for data2, got matched=%v, err=%v", matched, err)
	}

	// Case 3: destinationProcessName 不匹配 (第二个 AND 里的条件失败)
	data3 := map[string]interface{}{
		"sourceProcessName":      "HaozipSvc.exe",
		"destinationProcessName": "calc.exe",
		"filePath":               "NT AUTHORITY\\SYSTEM",
	}
	matched, err = Match(data3, rule)
	if !matched && err == nil {
		// Expected
	} else {
		t.Errorf("Expected no match for data3, got matched=%v, err=%v", matched, err)
	}

	// Case 4: filePath 不匹配 (第三个 AND 里的 OR 失败)
	data4 := map[string]interface{}{
		"sourceProcessName":      "HaozipSvc.exe",
		"destinationProcessName": "C:\\Windows\\System32\\lsass.exe",
		"filePath":               "C:\\Users\\Guest\\file.txt",
	}
	matched, err = Match(data4, rule)
	if !matched && err == nil {
		// Expected
	} else {
		t.Errorf("Expected no match for data4, got matched=%v, err=%v", matched, err)
	}
}

func TestMatch_Operators(t *testing.T) {
	tests := []struct {
		name     string
		ruleJSON string
		data     map[string]interface{}
		want     bool
	}{
		{
			name:     "equals true",
			ruleJSON: `{"field": "a", "operator": "equals", "value": "test"}`,
			data:     map[string]interface{}{"a": "test"},
			want:     true,
		},
		{
			name:     "equals false",
			ruleJSON: `{"field": "a", "operator": "equals", "value": "test"}`,
			data:     map[string]interface{}{"a": "other"},
			want:     false,
		},
		{
			name:     "nested field access",
			ruleJSON: `{"field": "meta.os", "operator": "equals", "value": "linux"}`,
			data:     map[string]interface{}{"meta": map[string]interface{}{"os": "linux"}},
			want:     true,
		},
		{
			name:     "cidr match",
			ruleJSON: `{"field": "ip", "operator": "cidr", "value": "192.168.1.0/24"}`,
			data:     map[string]interface{}{"ip": "192.168.1.100"},
			want:     true,
		},
		{
			name:     "greater than number",
			ruleJSON: `{"field": "count", "operator": "greater_than", "value": 10}`,
			data:     map[string]interface{}{"count": 20},
			want:     true,
		},
		{
			name:     "like match",
			ruleJSON: `{"field": "name", "operator": "like", "value": "test_%_prod"}`,
			data:     map[string]interface{}{"name": "test_web_prod"},
			want:     true,
		},
		{
			name:     "exists true",
			ruleJSON: `{"field": "optional", "operator": "exists", "value": null}`,
			data:     map[string]interface{}{"optional": "something"},
			want:     true,
		},
		{
			name:     "exists false",
			ruleJSON: `{"field": "missing", "operator": "exists", "value": null}`,
			data:     map[string]interface{}{"other": "val"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rule MatchRule
			if err := json.Unmarshal([]byte(tt.ruleJSON), &rule); err != nil {
				t.Fatalf("Failed to parse rule: %v", err)
			}
			got, err := Match(tt.data, rule)
			if err != nil {
				t.Errorf("Match() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}
