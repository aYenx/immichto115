package open115

import (
	"context"
	"fmt"
	"strings"

	"github.com/aYenx/immichto115/internal/config"
)

// Service 封装 115 Open 的配置读取与基础状态判断。
//
// 当前阶段先提供最小骨架，后续再接入 115-sdk-go：
// - 初始化真实 client
// - 自动刷新 token
// - 回写 access/refresh token
// - 用户信息查询 / 目录浏览 / 上传
//
// 这样做的目的是先把项目结构稳定下来，避免后面 API、授权、上传逻辑散落到 router 中。
type Service struct {
	cfg *config.Manager
}

func NewService(cfg *config.Manager) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Config() config.Open115Config {
	if s == nil || s.cfg == nil {
		return config.Open115Config{}
	}
	return s.cfg.Get().Open115
}

func (s *Service) TokenState() TokenState {
	cfg := s.Config()
	return TokenState{
		Enabled:        cfg.Enabled,
		ClientID:       cfg.ClientID,
		AccessToken:    cfg.AccessToken,
		RefreshToken:   cfg.RefreshToken,
		RootID:         cfg.RootID,
		TokenExpiresAt: cfg.TokenExpiresAt,
		UserID:         cfg.UserID,
	}
}

func (s *Service) HasUsableToken() bool {
	cfg := s.Config()
	return strings.TrimSpace(cfg.AccessToken) != "" && strings.TrimSpace(cfg.RefreshToken) != ""
}

// TestConnection 先做最小校验：检查 token 是否已配置。
// 后续接入 115-sdk-go 后，这里会改成实际调用用户信息接口验证 token 可用性。
func (s *Service) TestConnection(ctx context.Context) error {
	_ = ctx
	if s == nil || s.cfg == nil {
		return fmt.Errorf("open115 service not initialized")
	}
	if !s.HasUsableToken() {
		return fmt.Errorf("open115 access_token / refresh_token 未配置")
	}
	return nil
}
