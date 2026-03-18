package open115

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/aYenx/immichto115/internal/config"
	sdk "github.com/xhofe/115-sdk-go"
)

// Service 封装 115 Open 的配置读取、客户端初始化与 token 刷新回写。
type Service struct {
	cfg *config.Manager

	mu     sync.Mutex
	client Open115API
}

func NewService(cfg *config.Manager) *Service {
	return &Service{cfg: cfg}
}

// SetClient 允许注入自定义的 Open115API 实现（通常用于测试）。
func (s *Service) SetClient(c Open115API) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = c
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

func (s *Service) SafeTokenState() SafeTokenState {
	cfg := s.Config()
	return SafeTokenState{
		Enabled:        cfg.Enabled,
		ClientID:       cfg.ClientID,
		RootID:         cfg.RootID,
		TokenExpiresAt: cfg.TokenExpiresAt,
		UserID:         cfg.UserID,
	}
}

func (s *Service) HasUsableToken() bool {
	cfg := s.Config()
	return strings.TrimSpace(cfg.AccessToken) != "" && strings.TrimSpace(cfg.RefreshToken) != ""
}

func (s *Service) buildClient() (Open115API, error) {
	if s == nil || s.cfg == nil {
		return nil, fmt.Errorf("open115 service not initialized")
	}
	cfg := s.cfg.Get()
	openCfg := cfg.Open115
	if strings.TrimSpace(openCfg.AccessToken) == "" || strings.TrimSpace(openCfg.RefreshToken) == "" {
		return nil, fmt.Errorf("open115 access_token / refresh_token 未配置")
	}

	raw := sdk.New(
		sdk.WithAccessToken(openCfg.AccessToken),
		sdk.WithRefreshToken(openCfg.RefreshToken),
		sdk.WithOnRefreshToken(func(accessToken string, refreshToken string) {
			updated := s.cfg.Get()
			updated.Open115.AccessToken = accessToken
			updated.Open115.RefreshToken = refreshToken
			updated.Open115.Enabled = true
			if err := s.cfg.Update(updated); err != nil {
				// 当前阶段保持静默失败，避免在 SDK 回调里引发连锁错误。
				// 后续可替换为结构化日志。
			}
		}),
	)

	return newSDKClient(raw), nil
}

func (s *Service) Client() (Open115API, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s.client, nil
	}
	client, err := s.buildClient()
	if err != nil {
		return nil, err
	}
	s.client = client
	return s.client, nil
}

func (s *Service) ResetClient() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = nil
}

func (s *Service) updateUserState(userID int64) error {
	if s == nil || s.cfg == nil {
		return fmt.Errorf("open115 service not initialized")
	}
	updated := s.cfg.Get()
	updated.Open115.Enabled = true
	if userID > 0 {
		updated.Open115.UserID = strconv.FormatInt(userID, 10)
	}
	return s.cfg.Update(updated)
}

// TestConnection 使用 115 用户信息接口验证当前 token 是否可用。
func (s *Service) TestConnection(ctx context.Context) error {
	if s == nil || s.cfg == nil {
		return fmt.Errorf("open115 service not initialized")
	}
	client, err := s.Client()
	if err != nil {
		return err
	}
	pacer := NewPacer()
	user, err := Call(ctx, pacer, "UserInfo", defaultMaxRetries, func() (*UserInfoResp, error) {
		return client.UserInfo(ctx)
	})
	if err != nil {
		return err
	}
	if user == nil || user.UserID <= 0 {
		return fmt.Errorf("open115 用户信息为空")
	}
	if err := s.updateUserState(user.UserID); err != nil {
		return err
	}
	return nil
}

// TestConnectionDirect 使用传入的 token 验证连接，不依赖也不修改已保存的配置。
// 供设置页/向导页在用户点击"测试连接"时使用当前表单值而非后端旧值。
func TestConnectionDirect(ctx context.Context, accessToken, refreshToken string) error {
	if strings.TrimSpace(accessToken) == "" || strings.TrimSpace(refreshToken) == "" {
		return fmt.Errorf("access_token / refresh_token 不能为空")
	}
	api := newSDKClient(sdk.New(
		sdk.WithAccessToken(accessToken),
		sdk.WithRefreshToken(refreshToken),
	))
	pacer := NewPacer()
	user, err := Call(ctx, pacer, "UserInfo", defaultMaxRetries, func() (*UserInfoResp, error) {
		return api.UserInfo(ctx)
	})
	if err != nil {
		return err
	}
	if user == nil || user.UserID <= 0 {
		return fmt.Errorf("open115 用户信息为空")
	}
	return nil
}
