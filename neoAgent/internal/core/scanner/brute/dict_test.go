package brute

import (
	"reflect"
	"testing"
)

func TestDictManager_Generate(t *testing.T) {
	dm := NewDictManager()

	tests := []struct {
		name   string
		params map[string]interface{}
		mode   AuthMode
		want   []Auth
	}{
		{
			name:   "None Mode",
			params: nil,
			mode:   AuthModeNone,
			want:   []Auth{{}},
		},
		{
			name: "UserPass Mode with Custom Dict",
			params: map[string]interface{}{
				"users":     []string{"u1"},
				"passwords": []string{"p1", "%user%_123"},
			},
			mode: AuthModeUserPass,
			want: []Auth{
				{Username: "u1", Password: "p1"},
				{Username: "u1", Password: "u1_123"},
			},
		},
		{
			name: "UserPass Mode with Comma String Params",
			params: map[string]interface{}{
				"users":     "admin, root",
				"passwords": "123",
			},
			mode: AuthModeUserPass,
			want: []Auth{
				{Username: "admin", Password: "123"},
				{Username: "root", Password: "123"},
			},
		},
		{
			name: "OnlyPass Mode",
			params: map[string]interface{}{
				"passwords": []string{"pass", "%user%123"},
			},
			mode: AuthModeOnlyPass,
			want: []Auth{
				{Password: "pass"},
				{Password: "admin123"}, // Default fallback for %user% in OnlyPass
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dm.Generate(tt.params, tt.mode)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractStringSlice(t *testing.T) {
	// Test []interface{} which is common from JSON unmarshal
	m := map[string]interface{}{
		"users": []interface{}{"a", "b"},
	}
	got := extractStringSlice(m, "users", []string{"default"})
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractStringSlice with []interface{} = %v, want %v", got, want)
	}
}
