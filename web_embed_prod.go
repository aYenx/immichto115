//go:build embedfront

package immichto115

import "embed"

// WebDistFS embeds frontend build artifacts under web/dist.
// 构建前请先编译前端: cd web && npm ci && npm run build
// 然后使用: go build -tags embedfront ./cmd/server/
//
//go:embed all:web/dist
var WebDistFS embed.FS
