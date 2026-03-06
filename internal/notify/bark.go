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

// NotifyBackupResult 根据备份结果发送通知。
func NotifyBackupResult(barkURL string, success bool, detail string) {
	if barkURL == "" {
		return
	}

	var title, body string
	if success {
		title = "✅ 备份成功"
		body = "ImmichTo115 备份任务已完成"
		if detail != "" {
			body += "：" + detail
		}
	} else {
		title = "❌ 备份失败"
		body = "ImmichTo115 备份任务出错"
		if detail != "" {
			body += "：" + detail
		}
	}

	if err := SendBark(barkURL, title, body); err != nil {
		log.Printf("[notify] failed to send bark notification: %v", err)
	} else {
		log.Printf("[notify] bark notification sent: %s", title)
	}
}
