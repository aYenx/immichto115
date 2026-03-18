package api

import (
	"path/filepath"
	"regexp"
	"strings"
)

// sensitivePatterns matches strings that may leak internal details.
var sensitivePatterns = []*regexp.Regexp{
	// URL credentials: scheme://user:pass@host
	regexp.MustCompile(`://[^@/\s]+:[^@/\s]+@`),
	// Absolute Windows paths: C:\... or D:\...
	regexp.MustCompile(`[A-Za-z]:\\\\[^\s"']+`),
	// Absolute Unix paths: /home/... /etc/... /var/...
	regexp.MustCompile(`(?:^|\s)/(?:home|etc|var|usr|tmp|root|opt|srv|mnt|media)/[^\s"']*`),
	// Go stack trace lines: goroutine, .go:123
	regexp.MustCompile(`goroutine \d+`),
	regexp.MustCompile(`\S+\.go:\d+`),
}

// pathPrefixPattern detects long absolute paths in free-form error text.
var pathPrefixPattern = regexp.MustCompile(`(?:[A-Za-z]:[/\\]|/)(?:[^\s"':/]+[/\\]){2,}[^\s"']*`)

// sanitizeError scrubs potentially sensitive information from error messages
// before returning them in API responses. It removes:
//   - Embedded credentials in URLs
//   - Absolute file system paths (replaced with basenames)
//   - Go stack trace fragments
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	return SanitizeString(msg)
}

// SanitizeString scrubs a raw string, exported for testing.
func SanitizeString(msg string) string {
	// 1. Strip URL credentials
	msg = sensitivePatterns[0].ReplaceAllString(msg, "://***@")

	// 2. Replace deep absolute paths with basenames
	msg = pathPrefixPattern.ReplaceAllStringFunc(msg, func(p string) string {
		base := filepath.Base(strings.ReplaceAll(p, "\\", "/"))
		if base == "." || base == "/" {
			return p
		}
		return ".../" + base
	})

	// 3. Strip stack trace lines
	for _, pat := range sensitivePatterns[3:] {
		msg = pat.ReplaceAllString(msg, "[redacted]")
	}

	return strings.TrimSpace(msg)
}
