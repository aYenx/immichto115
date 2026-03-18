package notify

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendBark_EmptyURL(t *testing.T) {
	err := SendBark("", "title", "body")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestSendBark_POST_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/push" {
			t.Fatalf("expected /push, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := SendBark(srv.URL, "Test Title", "Test Body"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendBark_POST_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := SendBark(srv.URL, "Title", "Body")
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestFormatBackupTitle(t *testing.T) {
	tests := []struct {
		name string
		info BackupNotification
		want string
	}{
		{
			name: "success default trigger",
			info: BackupNotification{Success: true},
			want: "✅ 手动备份完成",
		},
		{
			name: "success with trigger",
			info: BackupNotification{Success: true, Trigger: "定时", Stage: "同步"},
			want: "✅ 定时同步完成",
		},
		{
			name: "failure default",
			info: BackupNotification{Success: false},
			want: "❌ 手动备份失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBackupTitle(tt.info)
			if got != tt.want {
				t.Fatalf("formatBackupTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyFailureReason(t *testing.T) {
	tests := []struct {
		detail string
		want   string
	}{
		{"", "请打开 ImmichTo115 实时日志查看详细报错"},
		{"任务已被手动停止", "这是一次手动停止，不是系统异常"},
		{"permission denied", "权限不足：请检查本地目录、WebDAV 目录或容器挂载权限"},
		{"connection refused", "连接失败：请检查 WebDAV 地址、端口、域名解析或服务是否在线"},
		{"timeout", "连接超时：请检查 WebDAV 服务是否可访问，或网络是否稳定"},
		{"random unknown error", "random unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.detail, func(t *testing.T) {
			got := classifyFailureReason(tt.detail)
			if got != tt.want {
				t.Fatalf("classifyFailureReason(%q) = %q, want %q", tt.detail, got, tt.want)
			}
		})
	}
}

func TestFormatBackupBody_SuccessContainsExpected(t *testing.T) {
	info := BackupNotification{
		Success:    true,
		Trigger:    "定时",
		Mode:       "增量备份",
		Stage:      "备份",
		RemotePath: "/backup/photos",
		Detail:     "上传了 5 个文件",
	}
	body := formatBackupBody(info)
	for _, want := range []string{"定时", "增量备份", "/backup/photos", "上传了 5 个文件", "已完成"} {
		if !contains(body, want) {
			t.Fatalf("formatBackupBody() missing %q in:\n%s", want, body)
		}
	}
}

func TestFormatBackupBody_FailureContainsReason(t *testing.T) {
	info := BackupNotification{
		Success: false,
		Detail:  "timeout",
	}
	body := formatBackupBody(info)
	if !contains(body, "超时") {
		t.Fatalf("expected classified reason in body, got:\n%s", body)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	return fmt.Sprintf("%s", s) != "" && len(sub) > 0 && stringContains(s, sub)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
