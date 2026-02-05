package brute_test

import (
	"reflect"
	"testing"

	"neoagent/internal/core/scanner/brute"
)

func TestDictManager_Generate(t *testing.T) {
	dm := brute.NewDictManager()

	tests := []struct {
		name   string
		params map[string]interface{}
		mode   brute.AuthMode
		want   []brute.Auth
	}{
		{
			name:   "None Mode",
			params: nil,
			mode:   brute.AuthModeNone,
			want:   []brute.Auth{{}},
		},
		{
			name: "UserPass Mode with Custom Dict",
			params: map[string]interface{}{
				"users":     []string{"u1"},
				"passwords": []string{"p1", "%user%_123"},
			},
			mode: brute.AuthModeUserPass,
			want: []brute.Auth{
				{Username: "u1", Password: "p1"},
				{Username: "u1", Password: "u1_123"},
			},
		},
		{
			name: "UserPass Mode with Comma String Params",
			params: map[string]interface{}{
				"users":           "admin, root",
				"passwords":       "123",
				"stop_on_success": true, // 添加这个参数以匹配实际行为
			},
			mode: brute.AuthModeUserPass,
			want: []brute.Auth{
				{Username: "admin", Password: "123"},
				{Username: "root", Password: "123"},
			},
		},
		{
			name: "OnlyPass Mode",
			params: map[string]interface{}{
				"passwords": []string{"pass", "%user%123"},
			},
			mode: brute.AuthModeOnlyPass,
			want: []brute.Auth{
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
