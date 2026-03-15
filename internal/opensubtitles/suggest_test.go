package opensubtitles

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sub/internal/model"
)

func newSuggestTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
		userAgent:  defaultUserAgent,
		maxRetries: 0,
		retryDelay: 1 * time.Millisecond,
	}
}

func TestSearchTitlesBuildsSuggestRequest(t *testing.T) {
	client := newSuggestTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/libs/suggest.php" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		if got := r.URL.Query().Get("format"); got != "json3" {
			t.Fatalf("expected format=json3, got %q", got)
		}

		if got := r.URL.Query().Get("MovieName"); got != "juju" {
			t.Fatalf("expected MovieName=juju, got %q", got)
		}

		if got := r.URL.Query().Get("SubLanguageID"); got != "eng" {
			t.Fatalf("expected SubLanguageID=eng, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[]`)
	})

	titles, err := client.SearchTitles(context.Background(), " juju ", "eng")
	if err != nil {
		t.Fatalf("expected search to succeed, got error: %v", err)
	}

	if len(titles) != 0 {
		t.Fatalf("expected no titles, got %d", len(titles))
	}
}

func TestSearchTitlesMapsTVKindToSeries(t *testing.T) {
	client := newSuggestTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[
			{"name":"Jujutsu Kaisen","year":"2020","id":976348,"pic":"12343534","kind":"tv","rating":"8.5"}
		]`)
	})

	titles, err := client.SearchTitles(context.Background(), "juju", "eng")
	if err != nil {
		t.Fatalf("expected search to succeed, got error: %v", err)
	}

	if len(titles) != 1 {
		t.Fatalf("expected 1 title, got %d", len(titles))
	}

	if titles[0].Type != model.TitleTypeSeries {
		t.Fatalf("expected result to be series, got %q", titles[0].Type)
	}
}

func TestSearchTitlesAcceptsNumericPicField(t *testing.T) {
	client := newSuggestTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[
			{"name":"Jujuba","year":"2019","id":1096211,"pic":0,"kind":"movie","rating":"0.0"}
		]`)
	})

	titles, err := client.SearchTitles(context.Background(), "juju", "eng")
	if err != nil {
		t.Fatalf("expected search to succeed, got error: %v", err)
	}

	if len(titles) != 1 {
		t.Fatalf("expected 1 title, got %d", len(titles))
	}

	if titles[0].IMDBID != 0 {
		t.Fatalf("expected numeric pic=0 to map to imdb id 0, got %d", titles[0].IMDBID)
	}
}

func TestSearchTitlesShortQuerySkipsRequest(t *testing.T) {
	called := false
	client := newSuggestTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[]`)
	})

	titles, err := client.SearchTitles(context.Background(), " j ", "eng")
	if err != nil {
		t.Fatalf("expected search to succeed, got error: %v", err)
	}

	if called {
		t.Fatal("expected no request for short query")
	}

	if len(titles) != 0 {
		t.Fatalf("expected no titles, got %d", len(titles))
	}
}
