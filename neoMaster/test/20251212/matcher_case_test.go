package matcher_test

import (
	"testing"

	"neomaster/internal/pkg/matcher"
)

func TestMatcherIgnoreCase(t *testing.T) {
	// 1. Test Equals IgnoreCase
	t.Run("Equals IgnoreCase", func(t *testing.T) {
		data := map[string]interface{}{"os": "Windows 10"}
		rule := matcher.MatchRule{
			Field:      "os",
			Operator:   "equals",
			Value:      "windows 10",
			IgnoreCase: true,
		}
		matched, err := matcher.Match(data, rule)
		if err != nil || !matched {
			t.Errorf("Should match 'Windows 10' with 'windows 10' when IgnoreCase=true")
		}
	})

	// 2. Test Contains IgnoreCase
	t.Run("Contains IgnoreCase", func(t *testing.T) {
		data := map[string]interface{}{"log": "Error: Connection Failed"}
		rule := matcher.MatchRule{
			Field:      "log",
			Operator:   "contains",
			Value:      "connection",
			IgnoreCase: true,
		}
		matched, err := matcher.Match(data, rule)
		if err != nil || !matched {
			t.Errorf("Should match 'Connection' with 'connection' when IgnoreCase=true")
		}
	})

	// 3. Test List Contains IgnoreCase
	t.Run("List Contains IgnoreCase", func(t *testing.T) {
		data := map[string]interface{}{"tags": []string{"Prod", "Cloud"}}
		rule := matcher.MatchRule{
			Field:      "tags",
			Operator:   "list_contains",
			Value:      "prod",
			IgnoreCase: true,
		}
		matched, err := matcher.Match(data, rule)
		if err != nil || !matched {
			t.Errorf("Should match 'Prod' with 'prod' in list when IgnoreCase=true")
		}
	})

	// 4. Test In IgnoreCase
	t.Run("In IgnoreCase", func(t *testing.T) {
		data := map[string]interface{}{"role": "Admin"}
		rule := matcher.MatchRule{
			Field:      "role",
			Operator:   "in",
			Value:      []string{"user", "admin", "guest"},
			IgnoreCase: true,
		}
		matched, err := matcher.Match(data, rule)
		if err != nil || !matched {
			t.Errorf("Should match 'Admin' in ['admin'] when IgnoreCase=true")
		}
	})

	// 5. Test Regex IgnoreCase
	t.Run("Regex IgnoreCase", func(t *testing.T) {
		data := map[string]interface{}{"email": "USER@Example.com"}
		rule := matcher.MatchRule{
			Field:      "email",
			Operator:   "regex",
			Value:      "^user@.*",
			IgnoreCase: true,
		}
		matched, err := matcher.Match(data, rule)
		if err != nil || !matched {
			t.Errorf("Should match 'USER@...' with '^user@.*' when IgnoreCase=true")
		}
	})

	// 6. Test Greater Than String Fallback IgnoreCase
	t.Run("Greater Than String IgnoreCase", func(t *testing.T) {
		// "a" > "B" is true in ASCII (97 > 66).
		// But if IgnoreCase, "a" vs "b". "a" < "b". So "a" > "b" is false.
		
		// Let's try "B" > "a". 
		// ASCII: "B" (66) < "a" (97). False.
		// IgnoreCase: "b" > "a". True.
		
		data := map[string]interface{}{"val": "B"}
		rule := matcher.MatchRule{
			Field:      "val",
			Operator:   "greater_than",
			Value:      "a",
			IgnoreCase: true,
		}
		matched, err := matcher.Match(data, rule)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if !matched {
			t.Errorf("Expected 'B' > 'a' when IgnoreCase=true (because 'b' > 'a')")
		}
	})
}
