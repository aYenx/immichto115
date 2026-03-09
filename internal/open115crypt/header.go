package open115crypt

type FileHeader struct {
	Magic         string `json:"magic"`
	Version       string `json:"version"`
	Algorithm     string `json:"algorithm"`
	OriginalName  string `json:"original_name"`
	OriginalSize  int64  `json:"original_size"`
	OriginalMTime int64  `json:"original_mtime"`
}
