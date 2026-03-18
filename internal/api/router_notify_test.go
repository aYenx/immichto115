package api

import (
	"strings"
	"testing"
	"time"

	"github.com/aYenx/immichto115/internal/notify"
)

func TestBuildFailureNotification(t *testing.T) {
	n := buildFailureNotification(
		"备份", "定时任务", "增量备份（copy）", "照片库备份",
		"timeout", []string{"数据库备份"}, "/backup",
	)

	if n.Success {
		t.Fatal("expected Success=false")
	}
	if n.TaskType != "备份" {
		t.Fatalf("TaskType = %q, want 备份", n.TaskType)
	}
	if n.Trigger != "定时任务" {
		t.Fatalf("Trigger = %q, want 定时任务", n.Trigger)
	}
	if n.Mode != "增量备份（copy）" {
		t.Fatalf("Mode = %q, want 增量备份（copy）", n.Mode)
	}
	if n.FailedStage != "照片库备份" {
		t.Fatalf("FailedStage = %q, want 照片库备份", n.FailedStage)
	}
	if n.Reason != "timeout" {
		t.Fatalf("Reason = %q, want timeout", n.Reason)
	}
	if len(n.CompletedStages) != 1 || n.CompletedStages[0] != "数据库备份" {
		t.Fatalf("CompletedStages = %v, want [数据库备份]", n.CompletedStages)
	}
	if n.RemotePath != "/backup" {
		t.Fatalf("RemotePath = %q, want /backup", n.RemotePath)
	}
	if n.CompletedAt.IsZero() {
		t.Fatal("CompletedAt should not be zero")
	}
	if n.Summary != "" {
		t.Fatalf("Summary should be empty for failure, got %q", n.Summary)
	}
}

func TestBuildSuccessNotification(t *testing.T) {
	n := buildSuccessNotification(
		"摄影上传", "摄影上传", "摄影文件上传",
		"扫描 10，上传 8，失败 0，删除 8", "/photos",
	)

	if !n.Success {
		t.Fatal("expected Success=true")
	}
	if n.TaskType != "摄影上传" {
		t.Fatalf("TaskType = %q, want 摄影上传", n.TaskType)
	}
	if n.Summary != "扫描 10，上传 8，失败 0，删除 8" {
		t.Fatalf("Summary = %q", n.Summary)
	}
	if n.CompletedAt.IsZero() {
		t.Fatal("CompletedAt should not be zero")
	}
	if n.FailedStage != "" {
		t.Fatalf("FailedStage should be empty for success, got %q", n.FailedStage)
	}
	if n.Reason != "" {
		t.Fatalf("Reason should be empty for success, got %q", n.Reason)
	}
}

func TestBuildFailureNotification_PhotoUploadNoRemotePath(t *testing.T) {
	n := buildFailureNotification(
		"摄影上传", "摄影上传", "摄影文件上传", "摄影上传",
		"任务已被手动停止", nil, "",
	)

	if n.RemotePath != "" {
		t.Fatalf("RemotePath should be empty, got %q", n.RemotePath)
	}
	if n.CompletedStages != nil {
		t.Fatalf("CompletedStages should be nil, got %v", n.CompletedStages)
	}
}

func TestBuildNotification_FormatsCorrectly(t *testing.T) {
	// Verify the built notification produces correct title/body
	// when passed through the notify formatter.
	orig := notify.NowFuncForTest()
	fixedTime := time.Date(2026, 3, 19, 1, 30, 0, 0, time.Local)
	notify.SetNowFuncForTest(func() time.Time { return fixedTime })
	defer notify.SetNowFuncForTest(orig)

	n := buildFailureNotification(
		"备份", "定时任务", "增量备份（copy）", "照片库备份",
		"connection refused", []string{"数据库备份"}, "/backup",
	)
	n.CompletedAt = fixedTime

	title, body := notify.FormatNotification(n)

	if !strings.Contains(title, "❌") {
		t.Fatalf("title should contain failure emoji, got: %s", title)
	}
	if !strings.Contains(title, "照片库备份") {
		t.Fatalf("title should contain failed stage, got: %s", title)
	}
	if !strings.Contains(body, "📊 已完成：数据库备份") {
		t.Fatalf("body should contain completed stages, got:\n%s", body)
	}
	if !strings.Contains(body, "连接失败") {
		t.Fatalf("body should contain classified reason, got:\n%s", body)
	}
}
