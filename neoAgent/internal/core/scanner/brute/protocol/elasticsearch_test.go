package protocol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"neoagent/internal/core/scanner/brute"
)

func TestElasticsearchCracker_Check(t *testing.T) {
	// 模拟 Elasticsearch 服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 URL
		if r.URL.Path != "/_security/_authenticate" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// 验证 Basic Auth
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if user == "elastic" && pass == "password" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"username":"elastic","roles":["superuser"]}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer ts.Close()

	// 解析测试服务器的地址
	// ts.URL 格式如 http://127.0.0.1:12345
	host := strings.TrimPrefix(ts.URL, "http://")
	parts := strings.Split(host, ":")
	_ = parts[0] // serverHost 未被使用
	_ = 9200     // serverPort 未被使用
	if len(parts) > 1 {
		// 忽略错误，测试代码简单处理
		portVal := 0
		fmtSscan(parts[1], &portVal)
		_ = portVal // serverPort
	}

	cracker := NewElasticsearchCracker()

	tests := []struct {
		name    string
		auth    brute.Auth
		want    bool
		wantErr bool
	}{
		{
			name:    "Success",
			auth:    brute.Auth{Username: "elastic", Password: "password"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "Failure",
			auth:    brute.Auth{Username: "elastic", Password: "wrong"},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// 需要从 ts.URL 解析端口
			// 简单起见，我们直接 hack 一下 checkProtocol 或者 传递真实端口
			// 由于 Check 接口限制了 host 和 port，我们需要解析 ts.URL

			// 重新解析 ts.URL
			// http://127.0.0.1:55555
			urlParts := strings.Split(ts.URL, ":")
			portStr := urlParts[2]
			var p int
			// 简单的 string to int
			for _, c := range portStr {
				p = p*10 + int(c-'0')
			}

			got, err := cracker.Check(ctx, "127.0.0.1", p, tt.auth)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 辅助函数：解析端口
func fmtSscan(str string, v *int) {
	// 简单实现，测试用
	var res int
	for _, c := range str {
		if c >= '0' && c <= '9' {
			res = res*10 + int(c-'0')
		}
	}
	*v = res
}
