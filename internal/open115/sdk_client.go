package open115

import (
	"context"

	sdk "github.com/xhofe/115-sdk-go"
)

// sdkClient 是 Open115API 接口的真实实现，委托给 *sdk.Client。
type sdkClient struct {
	raw *sdk.Client
}

// newSDKClient 将一个 *sdk.Client 包装为 Open115API。
func newSDKClient(c *sdk.Client) Open115API {
	return &sdkClient{raw: c}
}

func (c *sdkClient) GetFiles(ctx context.Context, req *GetFilesReq) (*GetFilesResp, error) {
	return c.raw.GetFiles(ctx, req)
}

func (c *sdkClient) UploadInit(ctx context.Context, req *UploadInitReq) (*UploadInitResp, error) {
	return c.raw.UploadInit(ctx, req)
}

func (c *sdkClient) UploadGetToken(ctx context.Context) (*UploadGetTokenResp, error) {
	return c.raw.UploadGetToken(ctx)
}

func (c *sdkClient) Mkdir(ctx context.Context, parentID string, name string) (*MkdirResp, error) {
	return c.raw.Mkdir(ctx, parentID, name)
}

func (c *sdkClient) DelFile(ctx context.Context, req *DelFileReq) ([]string, error) {
	return c.raw.DelFile(ctx, req)
}

func (c *sdkClient) UserInfo(ctx context.Context) (*UserInfoResp, error) {
	return c.raw.UserInfo(ctx)
}

func (c *sdkClient) AuthDeviceCode(ctx context.Context, clientID string, codeVerifier string) (*AuthDeviceCodeResp, error) {
	return c.raw.AuthDeviceCode(ctx, clientID, codeVerifier)
}

func (c *sdkClient) QrCodeStatus(ctx context.Context, uid string, time string, sign string) (*QrCodeStatusResp, error) {
	return c.raw.QrCodeStatus(ctx, uid, time, sign)
}

func (c *sdkClient) CodeToToken(ctx context.Context, uid string, codeVerifier string) (*CodeToTokenResp, error) {
	return c.raw.CodeToToken(ctx, uid, codeVerifier)
}

func (c *sdkClient) DownURL(ctx context.Context, pickCode string, ua string) (DownURLResp, error) {
	return c.raw.DownURL(ctx, pickCode, ua)
}
