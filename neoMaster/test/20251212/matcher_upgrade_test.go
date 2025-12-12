package test_20251212

import (
	"testing"

	"neomaster/internal/pkg/matcher"
)

func TestMatcherUpgrade(t *testing.T) {
	// 1. Test list_contains
	t.Run("list_contains", func(t *testing.T) {
		data := map[string]interface{}{
			"tags": []string{"production", "cloud"},
		}

		// Match case
		ruleMatch := matcher.MatchRule{
			Field:    "tags",
			Operator: "list_contains",
			Value:    "production",
		}
		matched, err := matcher.Match(data, ruleMatch)
		if err != nil {
			t.Fatalf("list_contains match error: %v", err)
		}
		if !matched {
			t.Errorf("list_contains should match 'production' in tags")
		}

		// No match case
		ruleNoMatch := matcher.MatchRule{
			Field:    "tags",
			Operator: "list_contains",
			Value:    "dev",
		}
		matched, err = matcher.Match(data, ruleNoMatch)
		if err != nil {
			t.Fatalf("list_contains no match error: %v", err)
		}
		if matched {
			t.Errorf("list_contains should NOT match 'dev' in tags")
		}

		// Not a list case
		dataNotList := map[string]interface{}{
			"tags": "not_a_list",
		}
		matched, err = matcher.Match(dataNotList, ruleMatch)
		if err != nil {
			t.Fatalf("list_contains error on non-list: %v", err)
		}
		if matched {
			t.Errorf("list_contains should NOT match on non-list data")
		}
	})

	// 2. Test String Comparison Fallback
	t.Run("String Comparison Fallback", func(t *testing.T) {
		data := map[string]interface{}{
			"time": "08:30",
		}

		// "08:30" > "08:00"
		ruleGt := matcher.MatchRule{
			Field:    "time",
			Operator: "greater_than",
			Value:    "08:00",
		}
		matched, err := matcher.Match(data, ruleGt)
		if err != nil {
			t.Fatalf("greater_than string error: %v", err)
		}
		if !matched {
			t.Errorf("expected '08:30' > '08:00'")
		}

		// "08:30" < "09:00"
		ruleLt := matcher.MatchRule{
			Field:    "time",
			Operator: "less_than",
			Value:    "09:00",
		}
		matched, err = matcher.Match(data, ruleLt)
		if err != nil {
			t.Fatalf("less_than string error: %v", err)
		}
		if !matched {
			t.Errorf("expected '08:30' < '09:00'")
		}

		// Mixed Type (Number vs String) -> Should fallback to string comparison ONLY if not convertible
		// Case A: "10" vs "2a" (one is not number)
		// String compare: "10" < "2a" ? '1' < '2' -> True
		dataMixed := map[string]interface{}{
			"count": 10,
		}
		ruleMixed := matcher.MatchRule{
			Field:    "count",
			Operator: "less_than",
			Value:    "2a",
		}
		matched, err = matcher.Match(dataMixed, ruleMixed)
		if err != nil {
			t.Fatalf("mixed type comparison error: %v", err)
		}
		if !matched {
			t.Errorf("expected '10' < '2a' (string fallback)")
		}
	})

	t.Run("Mixed Type Numeric Priority", func(t *testing.T) {
		// Test that if both are convertible to numbers, they are compared as numbers
		data := map[string]interface{}{
			"count": 10,
		}
		// "2" can be parsed as float.
		rule := matcher.MatchRule{
			Field:    "count",
			Operator: "greater_than",
			Value:    "2",
		}
		matched, err := matcher.Match(data, rule)
		if err != nil {
			t.Fatalf("numeric priority error: %v", err)
		}
		if !matched {
			t.Errorf("expected 10 > 2 (numeric comparison)")
		}

		// Test "10" vs "2" where one is string but convertible
		dataStr := map[string]interface{}{
			"count": "10",
		}
		ruleStr := matcher.MatchRule{
			Field:    "count",
			Operator: "greater_than",
			Value:    "2",
		}
		matched, err = matcher.Match(dataStr, ruleStr)
		if err != nil {
			t.Fatalf("numeric priority string error: %v", err)
		}
		if !matched {
			t.Errorf("expected '10' > '2' (numeric comparison)")
		}
	})
}
