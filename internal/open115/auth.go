package open115

import (
	"context"
	"fmt"
	"time"
)

// StartAuth 启动扫码授权流程。
//
// 当前阶段先保留骨架与返回结构，下一步再真正接入：
// - device code
// - qrcode status
// - code -> token
func (s *Service) StartAuth(ctx context.Context, clientID string) (*AuthSession, error) {
	_ = ctx
	if clientID == "" {
		return nil, fmt.Errorf("client_id 不能为空")
	}
	return &AuthSession{
		UID:       "",
		Time:      0,
		Sign:      "",
		QRCode:    "",
		CreatedAt: time.Now(),
	}, nil
}

// CheckAuthStatus 查询扫码状态。
func (s *Service) CheckAuthStatus(ctx context.Context, session *AuthSession) (*AuthStatus, error) {
	_ = ctx
	if session == nil {
		return nil, fmt.Errorf("auth session 不存在")
	}
	return &AuthStatus{
		Status:     0,
		Message:    "pending",
		Authorized: false,
	}, nil
}

// FinishAuth 完成扫码授权并换取 token。
func (s *Service) FinishAuth(ctx context.Context, session *AuthSession) (*TokenState, error) {
	_ = ctx
	if session == nil {
		return nil, fmt.Errorf("auth session 不存在")
	}
	state := s.TokenState()
	return &state, nil
}
