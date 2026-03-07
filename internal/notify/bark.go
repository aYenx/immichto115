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

	// 确保 URL 以 / 结尾后拼接路径
	base := strings.TrimRight(barkURL, "/")

	payload := BarkPayload{
		Title: title,
		Body:  body,
		Group: "ImmichTo115",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal bark payload: %w", err)
	}

	// 对标题和正文进行 URL 编码，避免中文/emoji/特殊字符导致请求失败
	encodedURL := base + "/" + url.PathEscape(title) + "/" + url.PathEscape(body)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(encodedURL)
	if err != nil {
		// 回退到 POST JSON
		log.Printf("[notify] GET failed, falling back to POST: %v", err)
		req, _ := http.NewRequest("POST", base+"/push", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err = client.Do(req)
		if err != nil {
			return fmt.Errorf("bark push failed: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bark returned status %d", resp.StatusCode)
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
		if detail := strings.TrimSpace(info.Detail); detail != "" {
			parts = append(parts, "失败原因："+detail)
		} else {
			parts = append(parts, "失败原因：请打开 ImmichTo115 实时日志查看详细报错")
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
