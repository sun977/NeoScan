package dict

import (
	"reflect"
	"testing"
)

func TestExpandLine_Classic_WithEXT(t *testing.T) {
	got := ExpandLine("/backup.%EXT%", []string{"php", "bak"}, ExtensionModeClassic)
	want := []string{"/backup.php", "/backup.bak"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandLine_Classic_NoEXT(t *testing.T) {
	got := ExpandLine("/admin", []string{"php"}, ExtensionModeClassic)
	want := []string{"/admin"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandLine_Force(t *testing.T) {
	got := ExpandLine("/admin", []string{"php", "html"}, ExtensionModeForce)
	want := []string{"/admin", "/admin.php", "/admin.html"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandLine_Force_WithEXT(t *testing.T) {
	// Force 模式下含 %EXT% 的行也正常替换（不重复追加原路径）
	got := ExpandLine("/backup.%EXT%", []string{"php", "bak"}, ExtensionModeForce)
	want := []string{"/backup.php", "/backup.bak"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandLine_EmptyExtensions(t *testing.T) {
	// extensions 为空：含 %EXT% 也原样返回
	got := ExpandLine("/backup.%EXT%", nil, ExtensionModeClassic)
	want := []string{"/backup.%EXT%"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	got2 := ExpandLine("/admin", nil, ExtensionModeForce)
	want2 := []string{"/admin"}
	if !reflect.DeepEqual(got2, want2) {
		t.Errorf("got %v, want %v", got2, want2)
	}
}

func TestExpandLine_MultipleTokens(t *testing.T) {
	// 路径中多个 %EXT% 都被替换
	got := ExpandLine("/a.%EXT%/b.%EXT%", []string{"php"}, ExtensionModeClassic)
	want := []string{"/a.php/b.php"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
