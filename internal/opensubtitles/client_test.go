package opensubtitles

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGetRejectsOversizedResponseBody(t *testing.T) {
	payload := bytes.Repeat([]byte("a"), maxBodyBytes+1)
	client := newSuggestTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})

	_, err := client.get(context.Background(), "/oversized", url.Values{})
	if err == nil {
		t.Fatal("expected oversized body error, got nil")
	}

	if !strings.Contains(err.Error(), "response body too large") {
		t.Fatalf("expected oversized body error, got %v", err)
	}
}
