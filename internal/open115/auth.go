package open115

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aYenx/immichto115/internal/config"
	sdk "github.com/xhofe/115-sdk-go"
)

func generateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成 code verifier 失败: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

func newAuthClient() *sdk.Client {
	return sdk.New()
}

// StartAuth 启动扫码授权流程。
func (s *Service) StartAuth(ctx context.Context, clientID string) (*AuthSession, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, fmt.Errorf("client_id 不能为空")
	}
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, err
	}
	client := newAuthClient()
	resp, err := client.AuthDeviceCode(ctx, strings.TrimSpace(clientID), codeVerifier)
	if err != nil {
		return nil, err
	}
	return &AuthSession{
		UID:          resp.UID,
		Time:         resp.Time,
		Sign:         resp.Sign,
		QRCode:       resp.QrCode,
		CodeVerifier: codeVerifier,
		CreatedAt:    time.Now(),
	}, nil
}

// CheckAuthStatus 查询扫码状态。
func (s *Service) CheckAuthStatus(ctx context.Context, session *AuthSession) (*AuthStatus, error) {
	if session == nil {
		return nil, fmt.Errorf("auth session 不存在")
	}
	if strings.TrimSpace(session.UID) == "" || session.Time <= 0 || strings.TrimSpace(session.Sign) == "" {
		return nil, fmt.Errorf("auth session 缺少必要字段")
	}
	client := newAuthClient()
	resp, err := client.QrCodeStatus(ctx, session.UID, fmt.Sprintf("%d", session.Time), session.Sign)
	if err != nil {
		return nil, err
	}
	status := &AuthStatus{
		Status:     resp.Status,
		Message:    strings.TrimSpace(resp.Msg),
		Authorized: false,
	}
	// 115 扫码状态约定里，2 通常表示已确认；这里先做最保守映射。
	if resp.Status == 2 {
		status.Authorized = true
		if status.Message == "" {
			status.Message = "authorized"
		}
	} else if status.Message == "" {
		status.Message = "pending"
	}
	return status, nil
}

// FinishAuth 完成扫码授权并换取 token。
func (s *Service) FinishAuth(ctx context.Context, session *AuthSession) (*TokenState, error) {
	if s == nil || s.cfg == nil {
		return nil, fmt.Errorf("open115 service not initialized")
	}
	if session == nil {
		return nil, fmt.Errorf("auth session 不存在")
	}
	if strings.TrimSpace(session.UID) == "" || strings.TrimSpace(session.CodeVerifier) == "" {
		return nil, fmt.Errorf("auth session 缺少换 token 所需字段")
	}

	client := newAuthClient()
	tokenResp, err := client.CodeToToken(ctx, session.UID, session.CodeVerifier)
	if err != nil {
		return nil, err
	}

	updated := s.cfg.Get()
	updated.Provider = "open115"
	updated.Open115.Enabled = true
	updated.Open115.ClientID = strings.TrimSpace(updated.Open115.ClientID)
	updated.Open115.AccessToken = tokenResp.AccessToken
	updated.Open115.RefreshToken = tokenResp.RefreshToken
	updated.Open115.TokenExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()
	if strings.TrimSpace(updated.Open115.RootID) == "" {
		updated.Open115.RootID = "0"
	}
	if err := s.cfg.Update(updated); err != nil {
		return nil, err
	}

	s.ResetClient()
	state := s.TokenState()
	if err := s.TestConnection(ctx); err != nil {
		return nil, err
	}
	state = s.TokenState()
	return &state, nil
}
