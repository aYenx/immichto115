package manifest

// FileRecord 表示一条本地文件与远端上传状态记录。
type FileRecord struct {
	Path              string
	Size              int64
	MTime             int64
	SHA1              string
	PreID             string
	RemoteFileID      string
	RemotePickCode    string
	LastUploadedAt    int64
	Deleted           bool
	Encrypted         bool
	EncryptedSize     int64
	RemotePath        string
	EncryptionVersion string
	ContentSHA256     string
	PendingDeleteAt   int64 // Unix timestamp; 0 = not pending
}
