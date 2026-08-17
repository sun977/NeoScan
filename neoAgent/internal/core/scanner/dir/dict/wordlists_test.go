package dict

import (
	"testing"
)

func TestLoadBuiltinWordlist(t *testing.T) {
	lines, err := LoadBuiltinWordlist()
	if err != nil {
		t.Fatalf("LoadBuiltinWordlist() error: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected non-empty wordlist, got 0 entries")
	}
	// 不含空行
	for i, line := range lines {
		if line == "" {
			t.Errorf("found empty line at index %d", i)
		}
	}
	// 不含 # 注释行
	for i, line := range lines {
		if len(line) > 0 && line[0] == '#' {
			t.Errorf("found comment line at index %d: %q", i, line)
		}
	}
	t.Logf("LoadBuiltinWordlist: loaded %d entries", len(lines))
}

func TestLoadCategoryWordlist(t *testing.T) {
	tests := []struct {
		name     string
		category string
	}{
		{"wordpress via subdir search", "wordpress"},
		{"spring via subdir search", "spring"},
		{"django via subdir search", "django"},
		{"direct path php/wordpress", "php/wordpress"},
		{"direct path java/spring", "java/spring"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, err := LoadCategoryWordlist(tt.category)
			if err != nil {
				t.Errorf("LoadCategoryWordlist(%q) error: %v", tt.category, err)
				return
			}
			if len(lines) == 0 {
				t.Errorf("LoadCategoryWordlist(%q): empty result", tt.category)
			}
			t.Logf("category=%q: %d entries", tt.category, len(lines))
		})
	}
}

func TestLoadCategoryWordlist_NotFound(t *testing.T) {
	_, err := LoadCategoryWordlist("nonexistent_category_xyz")
	if err == nil {
		t.Fatal("expected ErrCategoryNotFound, got nil")
	}
	if err != ErrCategoryNotFound {
		t.Fatalf("expected ErrCategoryNotFound, got: %v", err)
	}
}

func TestLoadBlacklist(t *testing.T) {
	// 403 黑名单必须包含常见敏感路径
	bl, err := LoadBlacklist(403)
	if err != nil {
		t.Fatalf("LoadBlacklist(403) error: %v", err)
	}
	if len(bl) == 0 {
		t.Fatal("expected non-empty 403 blacklist")
	}
	t.Logf("403 blacklist: %d entries", len(bl))

	// 验证 400 和 500 也可加载
	for _, code := range []int{400, 500} {
		bl2, err2 := LoadBlacklist(code)
		if err2 != nil {
			t.Errorf("LoadBlacklist(%d) error: %v", code, err2)
			continue
		}
		t.Logf("%d blacklist: %d entries", code, len(bl2))
	}
}

func TestLoadBlacklist_Unsupported(t *testing.T) {
	_, err := LoadBlacklist(999)
	if err == nil {
		t.Fatal("expected error for unsupported status code, got nil")
	}
}

func TestLoadUserAgents(t *testing.T) {
	agents, err := LoadUserAgents()
	if err != nil {
		t.Fatalf("LoadUserAgents() error: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("expected non-empty user-agents list")
	}
	t.Logf("LoadUserAgents: %d entries", len(agents))
}

func TestListCategories(t *testing.T) {
	categories := ListCategories()
	if len(categories) == 0 {
		t.Fatal("expected non-empty categories list")
	}
	t.Logf("ListCategories: %v", categories)

	// 验证已知类别存在
	found := make(map[string]bool)
	for _, c := range categories {
		found[c] = true
	}
	// 按子目录方式命名的条目
	for _, expected := range []string{"php/wordpress", "java/spring", "python/django"} {
		if !found[expected] {
			t.Errorf("expected category %q in list, not found. All: %v", expected, categories)
		}
	}
}
