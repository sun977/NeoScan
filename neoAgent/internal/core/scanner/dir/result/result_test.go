package result

import (
	"fmt"
	"testing"
	"time"
)

func TestDirHit_String(t *testing.T) {
	hit := &DirHit{
		Path:   "/admin",
		Status: 200,
		Size:   1024,
	}
	want := "[200] /admin (1024 bytes)"
	if hit.String() != want {
		t.Errorf("DirHit.String() = %q, want %q", hit.String(), want)
	}
}

func TestDirHit_String_MetricSize(t *testing.T) {
	hit := &DirHit{
		Path:   "/backup.zip",
		Status: 200,
		Size:   1572864, // 1.5 MB
	}
	got := hit.String()
	if got == "" {
		t.Fatal("expected non-empty string")
	}
	t.Logf("DirHit.String() = %q", got)
}

func TestDirResult_Add(t *testing.T) {
	r := NewDirResult("http://example.com", 1000)
	if r.Found != 0 {
		t.Errorf("expected Found=0 initially, got %d", r.Found)
	}
	if r.DictSize != 1000 {
		t.Errorf("expected DictSize=1000, got %d", r.DictSize)
	}
	if r.Hits == nil {
		t.Fatal("expected non-nil Hits slice")
	}

	hit := &DirHit{Path: "/admin", Status: 200, Size: 512}
	r.Add(hit)

	if r.Found != 1 {
		t.Errorf("expected Found=1 after Add, got %d", r.Found)
	}
	if len(r.Hits) != 1 {
		t.Errorf("expected len(Hits)=1, got %d", len(r.Hits))
	}
	if r.Hits[0] != hit {
		t.Error("expected Hits[0] to be the added hit")
	}
}

func TestDirResult_Finish(t *testing.T) {
	r := NewDirResult("http://example.com", 100)
	if r.StartTime.IsZero() {
		t.Error("expected non-zero StartTime after NewDirResult")
	}
	// 等待 10ms 确保 Finish 后的 EndTime 晚于 StartTime
	time.Sleep(10 * time.Millisecond)
	r.Finish()
	if r.EndTime.IsZero() {
		t.Fatal("expected non-zero EndTime after Finish")
	}
	if r.EndTime.Before(r.StartTime) {
		t.Error("EndTime should be >= StartTime")
	}
}

func TestDirResult_Headers(t *testing.T) {
	r := NewDirResult("http://example.com", 100)
	headers := r.Headers()
	want := []string{"Status", "Path", "Size", "Title", "Location"}
	if len(headers) != len(want) {
		t.Fatalf("Headers() = %v (%d items), want %v (%d items)", headers, len(headers), want, len(want))
	}
	for i, h := range want {
		if headers[i] != h {
			t.Errorf("Headers()[%d] = %q, want %q", i, headers[i], h)
		}
	}
}

func TestDirResult_Rows(t *testing.T) {
	r := NewDirResult("http://example.com", 100)

	// 添加几个不同的命中
	r.Add(&DirHit{Path: "/admin", Status: 200, Size: 1024, Title: "Admin Panel"})
	r.Add(&DirHit{Path: "/.env", Status: 403, Size: 0, Location: "/login"})
	r.Add(&DirHit{Path: "/backup.zip", Status: 200, Size: 5242880}) // 5 MB

	rows := r.Rows()
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// 第一行：200 /admin (1.0KB) Admin Panel -
	if rows[0][0] != "200" || rows[0][1] != "/admin" || rows[0][2] != "1.0KB" || rows[0][3] != "Admin Panel" {
		t.Errorf("row[0] = %v", rows[0])
	}
	// 第二行：403 /.env (0B) - /login
	if rows[1][0] != "403" || rows[1][1] != "/.env" || rows[1][3] != "-" || rows[1][4] != "/login" {
		t.Errorf("row[1] = %v", rows[1])
	}
	// 第三行：200 /backup.zip (5.0MB)
	if rows[2][0] != "200" || rows[2][2] != "5.0MB" {
		t.Errorf("row[2] = %v", rows[2])
	}
}

func TestDirResult_Rows_Empty(t *testing.T) {
	r := NewDirResult("http://example.com", 100)
	rows := r.Rows()
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty result, got %d", len(rows))
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size   int64
		expect string
	}{
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{10485760, "10.0MB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.size)
		if got != tt.expect {
			t.Errorf("formatSize(%d) = %q, want %q", tt.size, got, tt.expect)
		}
	}
}

func TestScanStats_JSON(t *testing.T) {
	stats := ScanStats{
		TotalRequests:  1000,
		SuccessfulReqs: 50,
		FilteredReqs:   900,
		WildcardReqs:   40,
		ErrorReqs:      10,
		AvgRTT:         150 * time.Millisecond,
		MaxRTT:         2 * time.Second,
		MinRTT:         10 * time.Millisecond,
	}

	data := fmt.Sprintf("%+v", stats)
	if data == "" {
		t.Error("expected non-empty ScanStats string")
	}
	t.Logf("ScanStats = %s", data)
}

func TestDirResult_CompleteFlow(t *testing.T) {
	// 模拟完整扫描流程
	r := NewDirResult("http://test.local:8080", 5000)
	r.Stats.TotalRequests = 5000
	r.Stats.SuccessfulReqs = 3
	r.Stats.FilteredReqs = 4900
	r.Stats.WildcardReqs = 90
	r.Stats.ErrorReqs = 7

	r.Add(&DirHit{Path: "/admin", Status: 200, Size: 2048, Title: "Admin Login"})
	r.Add(&DirHit{Path: "/api/v1/users", Status: 200, Size: 512})
	r.Add(&DirHit{Path: "/.env", Status: 200, Size: 128})

	r.Finish()

	if r.Found != 3 {
		t.Errorf("expected Found=3, got %d", r.Found)
	}
	if len(r.Hits) != 3 {
		t.Errorf("expected 3 hits, got %d", len(r.Hits))
	}

	rows := r.Rows()
	if len(rows) != 3 {
		t.Errorf("expected 3 output rows, got %d", len(rows))
	}

	// 验证 Headers 和 Rows 数量一致
	if len(r.Headers()) != len(rows[0]) {
		t.Errorf("Headers() has %d items, Rows()[0] has %d items", len(r.Headers()), len(rows[0]))
	}
}
