package notify

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

// fixedTime 用于所有格式化测试的固定时间。
var fixedTime = time.Date(2026, 3, 19, 1, 30, 0, 0, time.Local)

func setupFixedClock(t *testing.T) {
	t.Helper()
	orig := nowFunc
	nowFunc = func() time.Time { return fixedTime }
	t.Cleanup(func() { nowFunc = orig })
}

func TestFormatTitle(t *testing.T) {
	tests := []struct {
		name string
		info TaskNotification
		want string
	}{
		{
			name: "backup success",
			info: TaskNotification{Success: true, TaskType: "备份"},
			want: "✅ 备份完成",
		},
		{
			name: "backup success default tasktype",
			info: TaskNotification{Success: true},
			want: "✅ 备份完成",
		},
		{
			name: "photo upload success",
			info: TaskNotification{Success: true, TaskType: "摄影上传"},
			want: "✅ 摄影上传完成",
		},
		{
			name: "failure with failed stage",
			info: TaskNotification{Success: false, TaskType: "备份", FailedStage: "照片库备份"},
			want: "❌ 照片库备份失败",
		},
		{
			name: "failure with stage fallback",
			info: TaskNotification{Success: false, TaskType: "备份", Stage: "数据库备份"},
			want: "❌ 数据库备份失败",
		},
		{
			name: "failure default",
			info: TaskNotification{Success: false},
			want: "❌ 备份失败",
		},
		{
			name: "photo upload failure",
			info: TaskNotification{Success: false, TaskType: "摄影上传", FailedStage: "摄影上传"},
			want: "❌ 摄影上传失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTitle(tt.info)
			if got != tt.want {
				t.Fatalf("formatTitle() = %q, want %q", got, tt.want)
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

func TestFormatBody_Success(t *testing.T) {
	setupFixedClock(t)

	info := TaskNotification{
		Success:     true,
		TaskType:    "备份",
		Trigger:     "定时任务",
		Mode:        "增量备份（copy）",
		RemotePath:  "/backup/photos",
		Summary:     "扫描 120，上传 5，跳过 115",
		CompletedAt: fixedTime,
	}
	body := formatBody(info)

	for _, want := range []string{
		"📋 任务摘要",
		"━━━━━━━━━━━━━━",
		"⏰ 03-19 01:30",
		"🔄 定时任务 · 增量备份（copy）",
		"📂 /backup/photos",
		"✅ 任务完成",
		"扫描 120，上传 5，跳过 115",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("formatBody() missing %q in:\n%s", want, body)
		}
	}
}

func TestFormatBody_Failure(t *testing.T) {
	setupFixedClock(t)

	info := TaskNotification{
		Success:         false,
		TaskType:        "备份",
		Trigger:         "定时任务",
		Mode:            "增量备份（copy）",
		RemotePath:      "/backup",
		FailedStage:     "照片库备份",
		Reason:          "timeout",
		CompletedStages: []string{"数据库备份"},
		CompletedAt:     fixedTime,
	}
	body := formatBody(info)

	for _, want := range []string{
		"⏰ 03-19 01:30",
		"❌ 照片库备份失败",
		"超时", // classifyFailureReason result
		"📊 已完成：数据库备份",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("formatBody() missing %q in:\n%s", want, body)
		}
	}

	// 原始信息也应展示
	if !strings.Contains(body, "📎 原始信息：timeout") {
		t.Fatalf("formatBody() should contain original reason, got:\n%s", body)
	}
}

func TestFormatBody_FailureNoCompletedStages(t *testing.T) {
	setupFixedClock(t)

	info := TaskNotification{
		Success:     false,
		Reason:      "任务已被手动停止",
		CompletedAt: fixedTime,
	}
	body := formatBody(info)

	if !strings.Contains(body, "❌ 任务未完成") {
		t.Fatalf("expected generic failure header, got:\n%s", body)
	}
	if strings.Contains(body, "📊 已完成") {
		t.Fatalf("should not show completed stages when empty, got:\n%s", body)
	}
}

func TestFormatBody_PhotoUploadSuccess(t *testing.T) {
	setupFixedClock(t)

	info := TaskNotification{
		Success:     true,
		TaskType:    "摄影上传",
		Trigger:     "摄影上传",
		Mode:        "摄影文件上传",
		Summary:     "扫描 10，上传 8，失败 0，删除 8",
		CompletedAt: fixedTime,
	}
	body := formatBody(info)

	for _, want := range []string{
		"🔄 摄影上传 · 摄影文件上传",
		"✅ 任务完成",
		"扫描 10，上传 8",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("formatBody() missing %q in:\n%s", want, body)
		}
	}
}

func TestFormatBody_CompletedAtZeroFallback(t *testing.T) {
	setupFixedClock(t)

	info := TaskNotification{
		Success: true,
		// CompletedAt is zero value
	}
	body := formatBody(info)

	// Should fallback to nowFunc which returns fixedTime
	if !strings.Contains(body, "⏰ 03-19 01:30") {
		t.Fatalf("expected fallback timestamp, got:\n%s", body)
	}
}

func TestFormatTestNotification(t *testing.T) {
	title, body := FormatTestNotification(fixedTime)

	if !strings.Contains(title, "✅") {
		t.Fatalf("test notification title should indicate success, got: %s", title)
	}
	if !strings.Contains(title, "通知测试") {
		t.Fatalf("test notification title should contain task type, got: %s", title)
	}
	if !strings.Contains(body, "⏰ 03-19 01:30") {
		t.Fatalf("test notification body should contain timestamp, got:\n%s", body)
	}
	if !strings.Contains(body, "通知服务连接正常") {
		t.Fatalf("test notification body should confirm connectivity, got:\n%s", body)
	}
	if !strings.Contains(body, "📋 任务摘要") {
		t.Fatalf("test notification should use same layout as real notifications, got:\n%s", body)
	}
}
