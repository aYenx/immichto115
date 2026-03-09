//go:build !embedfront

package immichto115

import "embed"

// WebDistFS 在不使用 embedfront 标签时为空 FS。
// 此时 serveFrontend 会检测到 "web/dist" 不存在并跳过前端静态资源。
// 生产构建请使用: go build -tags embedfront ./cmd/server/
var WebDistFS embed.FS
