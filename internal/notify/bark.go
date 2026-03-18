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

// nowFunc 可注入的时钟，测试可替换为固定时间。
var nowFunc = time.Now

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

// TaskNotification 描述一次任务通知的关键信息。
type TaskNotification struct {
	Success         bool
	TaskType        string   // "备份" / "摄影上传"
	Trigger         string   // "手动" / "定时任务"
	Mode            string   // "增量备份（copy）" / "摄影文件上传"
	Stage           string   // 回退用笼统阶段名
	RemotePath      string
	CompletedStages []string // 已完成阶段
	FailedStage     string   // 失败的阶段名
	Reason          string   // 失败原因（原始 error）
	Summary         string   // 成功时摘要
	CompletedAt     time.Time
}

// BackupNotification 兼容别名，保持现有代码编译通过。
type BackupNotification = TaskNotification

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

// formatTitle 生成通知标题。
// 成功时：✅ {TaskType}完成
// 失败时：❌ {FailedStage|Stage|TaskType}失败
func formatTitle(info TaskNotification) string {
	taskType := fallbackText(info.TaskType, "备份")
	if info.Success {
		return "✅ " + taskType + "完成"
	}
	stage := strings.TrimSpace(info.FailedStage)
	if stage == "" {
		stage = fallbackText(info.Stage, taskType)
	}
	return "❌ " + stage + "失败"
}

// classifyFailureReason 将原始错误文本归类为更友好的故障描述。
func classifyFailureReason(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return "请打开 ImmichTo115 实时日志查看详细报错"
	}

	lower := strings.ToLower(detail)

	switch {
	case strings.Contains(detail, "任务已被手动停止"):
		return "这是一次手动停止，不是系统异常"
	case strings.Contains(lower, "already running") || strings.Contains(detail, "已有备份任务正在运行"):
		return "已有备份任务在运行，本次请求被跳过，请等待当前任务完成"
	case strings.Contains(lower, "permission denied") || strings.Contains(detail, "权限"):
		return "权限不足：请检查本地目录、WebDAV 目录或容器挂载权限"
	case strings.Contains(lower, "authentication") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden"):
		return "认证失败：请检查 WebDAV 用户名、密码或授权信息"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return "连接超时：请检查 WebDAV 服务是否可访问，或网络是否稳定"
	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host") || strings.Contains(lower, "dial tcp"):
		return "连接失败：请检查 WebDAV 地址、端口、域名解析或服务是否在线"
	case strings.Contains(lower, "directory not found") || strings.Contains(lower, "file does not exist") || strings.Contains(lower, "not found"):
		return "目录不存在：请检查本地备份路径或远端目录是否填写正确"
	case strings.Contains(lower, "config") || strings.Contains(detail, "未配置") || strings.Contains(detail, "生成 rclone 配置失败"):
		return "配置有误：请检查 WebDAV、远端目录、加密参数和备份路径是否填写完整"
	default:
		return detail
	}
}

// formatBody 生成通知正文。
func formatBody(info TaskNotification) string {
	ts := info.CompletedAt
	if ts.IsZero() {
		ts = nowFunc()
	}

	var b strings.Builder

	// 头部摘要区
	b.WriteString("📋 任务摘要\n")
	b.WriteString("━━━━━━━━━━━━━━\n")
	b.WriteString("⏰ " + ts.Format("01-02 15:04") + "\n")

	trigger := fallbackText(info.Trigger, "手动")
	mode := strings.TrimSpace(info.Mode)
	if mode != "" {
		b.WriteString("🔄 " + trigger + " · " + mode + "\n")
	} else {
		b.WriteString("🔄 " + trigger + "\n")
	}

	if remote := strings.TrimSpace(info.RemotePath); remote != "" {
		b.WriteString("📂 " + remote + "\n")
	}

	b.WriteString("\n")

	// 结果区
	if info.Success {
		b.WriteString("✅ 任务完成")
		if summary := strings.TrimSpace(info.Summary); summary != "" {
			b.WriteString("\n" + summary)
		}
	} else {
		// 失败标题行
		failedStage := strings.TrimSpace(info.FailedStage)
		if failedStage != "" {
			b.WriteString("❌ " + failedStage + "失败")
		} else {
			b.WriteString("❌ 任务未完成")
		}

		// 失败原因
		reason := classifyFailureReason(info.Reason)
		b.WriteString("\n" + reason)

		// 如分类结果与原始信息不同，附上原始信息
		if rawReason := strings.TrimSpace(info.Reason); rawReason != "" && reason != rawReason {
			b.WriteString("\n\n📎 原始信息：" + rawReason)
		}
	}

	// 已完成阶段（失败时显示，帮助用户了解进度）
	if !info.Success && len(info.CompletedStages) > 0 {
		b.WriteString("\n\n📊 已完成：" + strings.Join(info.CompletedStages, "、"))
	}

	return b.String()
}

// NotifyBackupResult 根据任务结果发送通知。
func NotifyBackupResult(barkURL string, info TaskNotification) {
	if barkURL == "" {
		return
	}

	title := formatTitle(info)
	body := formatBody(info)

	if err := SendBark(barkURL, title, body); err != nil {
		log.Printf("[notify] failed to send bark notification: %v", err)
	} else {
		log.Printf("[notify] bark notification sent: %s", title)
	}
}

// FormatTestNotification 生成测试通知的标题和正文。
// 复用同一套 formatTitle + formatBody 渲染逻辑，确保与真实通知排版一致。
func FormatTestNotification(at time.Time) (title, body string) {
	info := TaskNotification{
		Success:     true,
		TaskType:    "通知测试",
		Trigger:     "手动测试",
		Mode:        "",
		Summary:     "🔔 通知服务连接正常\n\n后续会推送任务结果，包括：\n• 成功 / 失败状态\n• 触发方式与任务模式\n• 失败原因与建议",
		CompletedAt: at,
	}
	return formatTitle(info), formatBody(info)
}

// FormatNotification 导出的格式化入口，返回标题和正文。
// 供外部包（如 api 测试）验证通知格式化结果。
func FormatNotification(info TaskNotification) (title, body string) {
	return formatTitle(info), formatBody(info)
}

// NowFuncForTest 返回当前的 nowFunc，供测试保存原始值。
func NowFuncForTest() func() time.Time {
	return nowFunc
}

// SetNowFuncForTest 设置 nowFunc，供测试注入固定时钟。返回原始值的 setter 以便 defer 还原。
func SetNowFuncForTest(fn func() time.Time) func() time.Time {
	old := nowFunc
	nowFunc = fn
	return old
}
