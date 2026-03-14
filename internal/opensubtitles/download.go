package opensubtitles

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const defaultArchiveName = "subtitle.zip"

func (c *Client) DownloadArchive(ctx context.Context, downloadURL string) (archive io.ReadCloser, fileName string, err error) {
	requestURL := resolveURL(c.baseURL, downloadURL)
	if requestURL == "" {
		return nil, "", fmt.Errorf("download url is empty")
	}

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		reader, resolvedName, shouldRetry, reqErr := c.downloadOnce(ctx, requestURL)
		if reqErr == nil {
			return reader, resolvedName, nil
		}

		lastErr = reqErr
		if !shouldRetry || attempt == c.maxRetries {
			break
		}

		delay := c.retryDelay * time.Duration(1<<attempt)
		if sleepErr := sleepWithContext(ctx, delay); sleepErr != nil {
			return nil, "", sleepErr
		}
	}

	return nil, "", lastErr
}

func (c *Client) downloadOnce(ctx context.Context, requestURL string) (archive io.ReadCloser, fileName string, shouldRetry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", false, fmt.Errorf("create download request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", false, fmt.Errorf("download request canceled: %w", ctx.Err())
		}

		return nil, "", true, fmt.Errorf("perform download request: %w", err)
	}

	if isTransientStatus(resp.StatusCode) {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, "", true, fmt.Errorf("transient status %d during download", resp.StatusCode)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, "", false, fmt.Errorf("unexpected download status %d: %s", resp.StatusCode, string(body))
	}

	resolvedName := archiveFileName(resp, requestURL)
	return resp.Body, resolvedName, false, nil
}

func archiveFileName(resp *http.Response, requestURL string) string {
	if resp != nil {
		raw := strings.TrimSpace(resp.Header.Get("Content-Disposition"))
		if raw != "" {
			_, params, err := mime.ParseMediaType(raw)
			if err == nil {
				if name := sanitizeFileName(params["filename"]); name != "" {
					return name
				}
				if name := sanitizeFileName(params["filename*"]); name != "" {
					return name
				}
			}
		}
	}

	u, err := url.Parse(requestURL)
	if err == nil {
		if name := sanitizeFileName(path.Base(u.Path)); name != "" {
			return name
		}
	}

	return defaultArchiveName
}

func sanitizeFileName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.TrimPrefix(trimmed, "UTF-8''")
	trimmed = strings.Trim(trimmed, `"'`)
	trimmed = path.Base(trimmed)
	if trimmed == "." || trimmed == "/" {
		return ""
	}

	trimmed = strings.ReplaceAll(trimmed, "\\", "_")
	trimmed = strings.ReplaceAll(trimmed, "/", "_")

	return strings.TrimSpace(trimmed)
}
