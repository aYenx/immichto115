package config

import (
	"path"
	"strings"
)

// CleanRemotePath normalizes a remote path:
//   - trims whitespace
//   - replaces backslashes with forward slashes
//   - ensures a leading "/"
//   - cleans redundant slashes and ".." segments
func CleanRemotePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	return path.Clean("/" + p)
}
