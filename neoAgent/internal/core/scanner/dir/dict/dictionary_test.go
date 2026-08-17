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
