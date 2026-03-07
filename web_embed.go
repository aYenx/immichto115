package immichto115

import "embed"

// WebDistFS embeds frontend build artifacts under web/dist.
// Build with: cd web && npm ci && npm run build
//go:embed all:web/dist
var WebDistFS embed.FS
