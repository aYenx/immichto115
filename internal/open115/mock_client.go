package open115

import "context"

// MockClient 是 Open115API 的测试替身。
// 通过设置各 XxxFunc 字段来控制返回值；未设置的方法会 panic。
type MockClient struct {
	GetFilesFunc       func(ctx context.Context, req *GetFilesReq) (*GetFilesResp, error)
	UploadInitFunc     func(ctx context.Context, req *UploadInitReq) (*UploadInitResp, error)
	UploadGetTokenFunc func(ctx context.Context) (*UploadGetTokenResp, error)
	MkdirFunc          func(ctx context.Context, parentID string, name string) (*MkdirResp, error)
	DelFileFunc        func(ctx context.Context, req *DelFileReq) ([]string, error)
	UserInfoFunc       func(ctx context.Context) (*UserInfoResp, error)
	AuthDeviceCodeFunc func(ctx context.Context, clientID string, codeVerifier string) (*AuthDeviceCodeResp, error)
	QrCodeStatusFunc   func(ctx context.Context, uid string, time string, sign string) (*QrCodeStatusResp, error)
	CodeToTokenFunc    func(ctx context.Context, uid string, codeVerifier string) (*CodeToTokenResp, error)
}

// compile-time check
var _ Open115API = (*MockClient)(nil)

func (m *MockClient) GetFiles(ctx context.Context, req *GetFilesReq) (*GetFilesResp, error) {
	return m.GetFilesFunc(ctx, req)
}

func (m *MockClient) UploadInit(ctx context.Context, req *UploadInitReq) (*UploadInitResp, error) {
	return m.UploadInitFunc(ctx, req)
}

func (m *MockClient) UploadGetToken(ctx context.Context) (*UploadGetTokenResp, error) {
	return m.UploadGetTokenFunc(ctx)
}

func (m *MockClient) Mkdir(ctx context.Context, parentID string, name string) (*MkdirResp, error) {
	return m.MkdirFunc(ctx, parentID, name)
}

func (m *MockClient) DelFile(ctx context.Context, req *DelFileReq) ([]string, error) {
	return m.DelFileFunc(ctx, req)
}

func (m *MockClient) UserInfo(ctx context.Context) (*UserInfoResp, error) {
	return m.UserInfoFunc(ctx)
}

func (m *MockClient) AuthDeviceCode(ctx context.Context, clientID string, codeVerifier string) (*AuthDeviceCodeResp, error) {
	return m.AuthDeviceCodeFunc(ctx, clientID, codeVerifier)
}

func (m *MockClient) QrCodeStatus(ctx context.Context, uid string, time string, sign string) (*QrCodeStatusResp, error) {
	return m.QrCodeStatusFunc(ctx, uid, time, sign)
}

func (m *MockClient) CodeToToken(ctx context.Context, uid string, codeVerifier string) (*CodeToTokenResp, error) {
	return m.CodeToTokenFunc(ctx, uid, codeVerifier)
}
