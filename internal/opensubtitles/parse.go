package opensubtitles

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func parseInt64(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}

	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}

	return v
}

func parseInt(raw string) int {
	return int(parseInt64(raw))
}

func parseFloat64(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}

	return v
}

func parseIDFromPath(path string, marker string) int64 {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0
	}

	idx := strings.Index(path, marker)
	if idx < 0 {
		return 0
	}

	start := idx + len(marker)
	if start >= len(path) {
		return 0
	}

	end := start
	for end < len(path) {
		ch := path[end]
		if ch < '0' || ch > '9' {
			break
		}
		end++
	}

	if end == start {
		return 0
	}

	return parseInt64(path[start:end])
}

func resolveURL(baseURL string, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err == nil && u.IsAbs() {
		return u.String()
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return raw
	}

	rel, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	return base.ResolveReference(rel).String()
}

func languagePath(language string) string {
	return fmt.Sprintf("sublanguageid-%s", normalizeLanguage(language))
}
