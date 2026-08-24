package dict

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDictionary_Next(t *testing.T) {
	opts := &DirOptions{}
	d, err := New(opts)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	count := 0
	for {
		path, ok := d.Next()
		if !ok {
			break
		}
		if path == "" {
			t.Error("Next() returned empty string with ok=true")
		}
		count++
	}

	if count == 0 {
		t.Fatal("expected at least 1 entry from builtin wordlist")
	}

	// 再次调用应返回 false
	_, ok := d.Next()
	if ok {
		t.Error("Next() after exhaustion should return false")
	}

	t.Logf("Dictionary.Next: iterated %d entries", count)
}

// TestDictionary_AddExtra_Dedup 验证 AddExtra 的 extra 队列内部去重：
// 重复路径跳过且不占用 total 名额（total 只统计唯一路径数）。
func TestDictionary_AddExtra_Dedup(t *testing.T) {
	d := &Dictionary{
		main:    []string{"/main-path"},
		maxSize: defaultMaxSize,
	}
	d.total = 1

	// 首次入队 + 重复入队混合
	d.AddExtra("/api/", "/admin/", "/api/", "/api/")
	d.AddExtra("/admin/", "/config/")

	// extra 队列应只有 3 条唯一路径
	if len(d.extra) != 3 {
		t.Errorf("extra len = %d, want 3 (dedup failed): %v", len(d.extra), d.extra)
	}

	// total = 1(main) + 3(唯一 extra)，重复路径不占名额
	if d.total != 4 {
		t.Errorf("total = %d, want 4 (1 main + 3 unique extra)", d.total)
	}

	// 消费顺序不受影响：extra 优先，按首次入队顺序
	want := []string{"/api/", "/admin/", "/config/", "/main-path"}
	for i, w := range want {
		got, ok := d.Next()
		if !ok || got != w {
			t.Errorf("Next()[%d] = %q (ok=%v), want %q", i, got, ok, w)
		}
	}
}

// TestDictionary_AddExtra_Dedup_MaxSize 验证字典接近满载时，
// 重复路径不再抢占 total 名额，真正的新路径不会被提前截断。
func TestDictionary_AddExtra_Dedup_MaxSize(t *testing.T) {
	d := &Dictionary{
		maxSize: 3,
	}
	d.total = 2 // 模拟已接近满载

	// 重复路径应被去重跳过，不占名额；/new-path 应成功入队
	d.AddExtra("/dup", "/dup", "/dup")
	if d.total != 3 {
		t.Errorf("total = %d, want 3 (only 1 unique path counted)", d.total)
	}

	// 此时已满载，新路径应被截断
	d.AddExtra("/overflow")
	if len(d.extra) != 1 {
		t.Errorf("extra len = %d, want 1 (overflow should be dropped)", len(d.extra))
	}
}

func TestDictionary_AddExtra_Priority(t *testing.T) {
	// 构造一个只有 1 条主字典的字典
	d := &Dictionary{
		main:    []string{"/main-path"},
		maxSize: defaultMaxSize,
	}
	d.total = 1

	d.AddExtra("/extra-1", "/extra-2")

	// 前两次应返回 extra
	first, ok := d.Next()
	if !ok || first != "/extra-1" {
		t.Errorf("expected /extra-1, got %q (ok=%v)", first, ok)
	}
	second, ok := d.Next()
	if !ok || second != "/extra-2" {
		t.Errorf("expected /extra-2, got %q (ok=%v)", second, ok)
	}
	// 第三次返回 main
	third, ok := d.Next()
	if !ok || third != "/main-path" {
		t.Errorf("expected /main-path, got %q (ok=%v)", third, ok)
	}
	// 耗尽
	_, ok = d.Next()
	if ok {
		t.Error("expected exhaustion after consuming all entries")
	}
}

func TestDictionary_MaxSize(t *testing.T) {
	// 构造一个 maxSize=5 的字典，加入 10 条条目，验证截断
	d := &Dictionary{
		maxSize: 5,
	}
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = fmt.Sprintf("/path-%d", i)
	}
	d.addToMain(lines)

	if len(d.main) > 5 {
		t.Errorf("expected at most 5 entries, got %d", len(d.main))
	}
}

func TestDictionary_Concurrent(t *testing.T) {
	opts := &DirOptions{}
	d, err := New(opts)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// 预先加几条 extra
	for i := 0; i < 100; i++ {
		d.AddExtra(fmt.Sprintf("/extra-%d", i))
	}

	var wg sync.WaitGroup
	results := make([]string, 0)
	var mu sync.Mutex

	// 100 个 goroutine 并发消费
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				path, ok := d.Next()
				if !ok {
					return
				}
				mu.Lock()
				results = append(results, path)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	t.Logf("Concurrent: consumed %d entries total", len(results))
}

// TestDictionary_CategoryWordlist 验证 New() 会加载 opts.Categories 指定的内置分类字典，
// 并追加到 main 队列（Bug 修复：LoadCategoryWordlist 此前无任何调用方）。
func TestDictionary_CategoryWordlist(t *testing.T) {
	d, err := New(&DirOptions{
		SkipBuiltin: true, // 跳过内置大字典，避免真实内置条目干扰断言
		Categories:  []string{"wordpress"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	wpLines, err := LoadCategoryWordlist("wordpress")
	if err != nil {
		t.Fatalf("LoadCategoryWordlist(wordpress) error: %v", err)
	}
	if len(wpLines) == 0 {
		t.Fatalf("LoadCategoryWordlist(wordpress) returned no entries")
	}
	if d.Total() != len(wpLines) {
		t.Errorf("Dictionary.Total() = %d, want %d (only wordpress category loaded)", d.Total(), len(wpLines))
	}

	first, ok := d.Next()
	if !ok || first != wpLines[0] {
		t.Errorf("Next() = %q, ok=%v, want %q", first, ok, wpLines[0])
	}
}

// TestDictionary_CategoryWordlist_NotFound 验证不存在的分类名不会导致 New() 报错，
// 仅记录警告并跳过（与 Wordlists 加载失败时的容错行为一致）。
func TestDictionary_CategoryWordlist_NotFound(t *testing.T) {
	d, err := New(&DirOptions{
		SkipBuiltin: true,
		Categories:  []string{"no-such-category"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if d.Total() != 0 {
		t.Errorf("Dictionary.Total() = %d, want 0 (invalid category should be skipped, not fatal)", d.Total())
	}
}

func TestDictionary_UserCustomWordlist(t *testing.T) {
	// 在临时目录创建模拟的 rules/dir/custom/ 结构
	tmpDir := t.TempDir()
	customDir := filepath.Join(tmpDir, "rules", "dir", "custom")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// 写入测试字典
	content := "/custom-path-1\n/custom-path-2\n# comment\n\n/custom-path-3\n"
	testFile := filepath.Join(customDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// 用绝对路径直接加载（避免依赖运行目录 fallback）
	lines, err := loadExternalFile(testFile)
	if err != nil {
		t.Fatalf("loadExternalFile: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines (comments and blanks filtered), got %d: %v", len(lines), lines)
	}
	expected := []string{"/custom-path-1", "/custom-path-2", "/custom-path-3"}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want)
		}
	}
}

func TestDictionary_ExtensionExpand(t *testing.T) {
	// 验证字典构建时扩展名正确展开
	d := &Dictionary{
		main:    []string{"/backup.%EXT%", "/admin"},
		maxSize: defaultMaxSize,
	}
	d.total = 2

	opts := &DirOptions{
		Extensions: []string{"php", "bak"},
		ExtMode:    ExtensionModeClassic,
	}
	d.expand(opts)

	// /backup.%EXT% → /backup.php, /backup.bak
	// /admin → /admin (Classic 模式不追加)
	if len(d.main) != 3 {
		t.Errorf("expected 3 entries after expansion, got %d: %v", len(d.main), d.main)
	}
}
