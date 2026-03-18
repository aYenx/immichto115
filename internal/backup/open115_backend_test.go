package backup

import (
	"context"
	"fmt"
	"testing"

	"github.com/aYenx/immichto115/internal/open115"
)

func TestOpen115Backend_Uploader(t *testing.T) {
	svc := open115.NewService(nil)
	backend := NewOpen115Backend(svc)
	if backend.Uploader() == nil {
		t.Fatal("expected non-nil Uploader()")
	}
}

func TestOpen115Backend_TestConnection_NoConfig(t *testing.T) {
	// Service without config → TestConnection should fail gracefully
	svc := open115.NewService(nil)
	backend := NewOpen115Backend(svc)
	err := backend.TestConnection(context.Background())
	if err == nil {
		t.Fatal("expected error when service has no config")
	}
}

func TestOpen115Backend_TestConnection_WithMock(t *testing.T) {
	mock := &open115.MockClient{
		UserInfoFunc: func(ctx context.Context) (*open115.UserInfoResp, error) {
			return nil, fmt.Errorf("mock auth failure")
		},
	}
	svc := open115.NewService(nil)
	svc.SetClient(mock)
	backend := NewOpen115Backend(svc)

	err := backend.TestConnection(context.Background())
	if err == nil {
		t.Fatal("expected error from mock")
	}
}

func TestOpen115Backend_EnsureDir_NilService(t *testing.T) {
	// Without proper setup, EnsureDir should fail
	svc := open115.NewService(nil)
	backend := NewOpen115Backend(svc)
	_, err := backend.EnsureDir(context.Background(), "/test/path")
	if err == nil {
		t.Fatal("expected error")
	}
}

