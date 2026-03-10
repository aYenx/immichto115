package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BarkPayload Bark 推送请求体。
type BarkPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Group string `json:"group,omitempty"`
	Icon  string `json:"icon,omitempty"`
}

// SendBark 发送 Bark 推送通知。
// barkURL 格式: https://api.day.app/YOUR_DEVICE_KEY
func SendBark(barkURL, title, body string) error {
	if barkURL == "" {
		return fmt.Errorf("bark URL is empty")
	}

	base := strings.TrimRight(barkURL, "/")

	payload := BarkPayload{
		Title: title,
		Body:  body,
		Group: "ImmichTo115",
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 优先使用 POST JSON（避免敏感信息出现在 URL / 服务端日志中）
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal bark payload: %w", err)
	}
	req, err := http.NewRequest("POST", base+"/push", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		return fmt.Errorf("bark returned status %d", resp.StatusCode)
	}

	// POST 失败，回退到 GET（兼容部分旧版 Bark 服务端）
	log.Printf("[notify] POST failed, falling back to GET: %v", err)
	encodedURL := base + "/" + url.PathEscape(title) + "/" + url.PathEscape(body)
	resp2, err := client.Get(encodedURL)
	if err != nil {
		return fmt.Errorf("bark push failed: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("bark returned status %d", resp2.StatusCode)
	}
	return nil
}

// BackupNotification 描述一次备份通知的关键信息。
type BackupNotification struct {
	Success    bool
	Trigger    string
	Mode       string
	Stage      string
	RemotePath string
	Detail     string
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func formatBackupTitle(info BackupNotification) string {
	trigger := fallbackText(info.Trigger, "手动")
	stage := fallbackText(info.Stage, "备份")
	if info.Success {
		return fmt.Sprintf("✅ %s%s完成", trigger, stage)
	}
	return fmt.Sprintf("❌ %s%s失败", trigger, stage)
}

func classifyFailureReason(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return "请打开 ImmichTo115 实时日志查看详细报错"
	}

	segments := strings.Split(detail, "；")
	core := detail
	if len(segments) > 0 {
		core = strings.TrimSpace(segments[len(segments)-1])
	}
	lower := strings.ToLower(core)

	switch {
	case strings.Contains(core, "任务已被手动停止"):
		return "这是一次手动停止，不是系统异常"
	case strings.Contains(lower, "already running") || strings.Contains(core, "已有备份任务正在运行"):
		return "已有备份任务在运行，本次请求被跳过，请等待当前任务完成"
	case strings.Contains(lower, "permission denied") || strings.Contains(core, "权限"):
		return "权限不足：请检查本地目录、WebDAV 目录或容器挂载权限"
	case strings.Contains(lower, "authentication") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden"):
		return "认证失败：请检查 WebDAV 用户名、密码或授权信息"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return "连接超时：请检查 WebDAV 服务是否可访问，或网络是否稳定"
	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host") || strings.Contains(lower, "dial tcp"):
		return "连接失败：请检查 WebDAV 地址、端口、域名解析或服务是否在线"
	case strings.Contains(lower, "directory not found") || strings.Contains(lower, "file does not exist") || strings.Contains(lower, "not found"):
		return "目录不存在：请检查本地备份路径或远端目录是否填写正确"
	case strings.Contains(lower, "config") || strings.Contains(core, "未配置") || strings.Contains(core, "生成 rclone 配置失败"):
		return "配置有误：请检查 WebDAV、远端目录、加密参数和备份路径是否填写完整"
	default:
		return detail
	}
}

func formatBackupBody(info BackupNotification) string {
	parts := []string{
		"应用：ImmichTo115",
		"触发方式：" + fallbackText(info.Trigger, "手动"),
		"备份模式：" + fallbackText(info.Mode, "增量备份"),
		"当前阶段：" + fallbackText(info.Stage, "备份"),
	}

	if remote := strings.TrimSpace(info.RemotePath); remote != "" {
		parts = append(parts, "远端位置："+remote)
	}

	if info.Success {
		parts = append(parts, "结果：本次任务已完成")
		if detail := strings.TrimSpace(info.Detail); detail != "" {
			parts = append(parts, "说明："+detail)
		}
	} else {
		parts = append(parts, "结果：本次任务未完成")
		parts = append(parts, "失败原因："+classifyFailureReason(info.Detail))
		if detail := strings.TrimSpace(info.Detail); detail != "" && classifyFailureReason(info.Detail) != detail {
			parts = append(parts, "原始信息："+detail)
		}
	}

	return strings.Join(parts, "\n")
}

// NotifyBackupResult 根据备份结果发送通知。
func NotifyBackupResult(barkURL string, info BackupNotification) {
	if barkURL == "" {
		return
	}

	title := formatBackupTitle(info)
	body := formatBackupBody(info)

	if err := SendBark(barkURL, title, body); err != nil {
		log.Printf("[notify] failed to send bark notification: %v", err)
	} else {
		log.Printf("[notify] bark notification sent: %s", title)
	}
}
