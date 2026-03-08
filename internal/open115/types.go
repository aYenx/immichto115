package open115

import "time"

// AuthSession 表示一次临时扫码授权会话。
type AuthSession struct {
	UID          string    `json:"uid"`
	Time         int64     `json:"time"`
	Sign         string    `json:"sign"`
	QRCode       string    `json:"qrcode"`
	CodeVerifier string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthStatus 表示扫码授权轮询结果。
type AuthStatus struct {
	Status     int    `json:"status"`
	Message    string `json:"message"`
	Authorized bool   `json:"authorized"`
}

// TokenState 表示当前保存的 115 Open token 状态。
type TokenState struct {
	Enabled        bool   `json:"enabled"`
	ClientID       string `json:"client_id"`
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	RootID         string `json:"root_id"`
	TokenExpiresAt int64  `json:"token_expires_at"`
	UserID         string `json:"user_id"`
}

// RemoteEntry 表示 Open115 远端目录项的通用结构。
type RemoteEntry struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size,omitempty"`
	ModTime  time.Time `json:"mod_time,omitempty"`
	PickCode string    `json:"pick_code,omitempty"`
}
