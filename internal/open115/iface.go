package open115

import (
	"context"

	sdk "github.com/xhofe/115-sdk-go"
)

// ----- 类型别名 -----
// 直接复用 SDK 类型以避免大规模签名变更。
// 未来如果需要彻底脱离 SDK 类型，只需在此处替换为自定义 struct。

type (
	GetFilesReq        = sdk.GetFilesReq
	GetFilesResp       = sdk.GetFilesResp
	GetFilesResp_File  = sdk.GetFilesResp_File
	UploadInitReq      = sdk.UploadInitReq
	UploadInitResp     = sdk.UploadInitResp
	UploadGetTokenResp = sdk.UploadGetTokenResp
	MkdirResp          = sdk.MkdirResp
	DelFileReq         = sdk.DelFileReq
	UserInfoResp       = sdk.UserInfoResp
	AuthDeviceCodeResp = sdk.AuthDeviceCodeResp
	QrCodeStatusResp   = sdk.QrCodeStatusResp
	CodeToTokenResp    = sdk.CodeToTokenResp
	DownURLResp        = sdk.DownURLResp
)

// Open115API 定义 115 Open SDK 的最小方法集。
// 所有对 SDK 的调用都应通过此接口，以便于测试和解耦。
type Open115API interface {
	// 文件列表
	GetFiles(ctx context.Context, req *GetFilesReq) (*GetFilesResp, error)

	// 上传初始化 / 秒传检查
	UploadInit(ctx context.Context, req *UploadInitReq) (*UploadInitResp, error)

	// 获取上传 STS Token
	UploadGetToken(ctx context.Context) (*UploadGetTokenResp, error)

	// 创建目录
	Mkdir(ctx context.Context, parentID string, name string) (*MkdirResp, error)

	// 删除文件/目录
	DelFile(ctx context.Context, req *DelFileReq) ([]string, error)

	// 用户信息
	UserInfo(ctx context.Context) (*UserInfoResp, error)

	// 扫码授权
	AuthDeviceCode(ctx context.Context, clientID string, codeVerifier string) (*AuthDeviceCodeResp, error)

	// 扫码状态轮询
	QrCodeStatus(ctx context.Context, uid string, time string, sign string) (*QrCodeStatusResp, error)

	// 换取 Token
	CodeToToken(ctx context.Context, uid string, codeVerifier string) (*CodeToTokenResp, error)

	// 获取文件下载链接
	DownURL(ctx context.Context, pickCode string, ua string) (DownURLResp, error)
}
