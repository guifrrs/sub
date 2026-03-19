package opensubtitles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sub/internal/model"
)

func TestSearchTitlesParsesSuggestFixture(t *testing.T) {
	fixture := loadFixture(t, "suggest_juju.json")

	client := newFixtureClient(t, map[string]string{
		"/libs/suggest.php": fixture,
	}, func(r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "json3" {
			t.Fatalf("expected format=json3, got %q", got)
		}
		if got := r.URL.Query().Get("MovieName"); got != "juju" {
			t.Fatalf("expected MovieName=juju, got %q", got)
		}
		if got := r.URL.Query().Get("SubLanguageID"); got != "eng" {
			t.Fatalf("expected SubLanguageID=eng, got %q", got)
		}
	})

	titles, err := client.SearchTitles(context.Background(), "juju", "eng")
	if err != nil {
		t.Fatalf("expected search to succeed, got: %v", err)
	}

	if len(titles) != 2 {
		t.Fatalf("expected 2 titles, got %d", len(titles))
	}

	if !titles[0].IsSeries() {
		t.Fatalf("expected first title to be series, got %q", titles[0].Type)
	}

	if titles[1].IMDBID != 0 {
		t.Fatalf("expected second title imdb id 0 from numeric pic, got %d", titles[1].IMDBID)
	}
}

func TestListEpisodesParsesSeriesFixture(t *testing.T) {
	fixture := loadFixture(t, "episodes_breaking_bad.xml")

	client := newFixtureClient(t, map[string]string{
		"/en/ssearch/sublanguageid-eng/idmovie-31305/xml": fixture,
	}, nil)

	series := model.Title{
		IDMovie: 31305,
		IMDBID:  903747,
		Name:    "Breaking Bad",
		Type:    model.TitleTypeSeries,
	}

	episodes, err := client.ListEpisodes(context.Background(), series, "eng")
	if err != nil {
		t.Fatalf("expected episode list to succeed, got: %v", err)
	}

	if len(episodes) != 3 {
		t.Fatalf("expected 3 unique episodes, got %d", len(episodes))
	}

	assertEpisode(t, episodes[0], 1, 1, 1000001)
	assertEpisode(t, episodes[1], 1, 2, 1000002)
	assertEpisode(t, episodes[2], 2, 1, 2000001)

	if episodes[0].SeriesName != "Breaking Bad" {
		t.Fatalf("expected series name propagation, got %q", episodes[0].SeriesName)
	}

	if episodes[0].SeriesIMDBID != 903747 {
		t.Fatalf("expected series imdb id propagation, got %d", episodes[0].SeriesIMDBID)
	}
}

func TestListSubtitlesByTitleParsesFixture(t *testing.T) {
	fixture := loadFixture(t, "subtitles_movie.xml")

	client := newFixtureClient(t, map[string]string{
		"/en/search/sublanguageid-eng/idmovie-6122/xml": fixture,
	}, nil)

	title := model.Title{IDMovie: 6122, Name: "Breakfast Club", Type: model.TitleTypeMovie}
	subtitles, err := client.ListSubtitlesByTitle(context.Background(), title, "eng")
	if err != nil {
		t.Fatalf("expected subtitle list to succeed, got: %v", err)
	}

	if len(subtitles) != 2 {
		t.Fatalf("expected 2 valid subtitles, got %d", len(subtitles))
	}

	if subtitles[0].DownloadURL != client.baseURL+"/en/download/subad/13130943" {
		t.Fatalf("expected relative link to resolve against base url, got %q", subtitles[0].DownloadURL)
	}

	if !subtitles[0].Trusted {
		t.Fatalf("expected first subtitle to be trusted")
	}

	if subtitles[0].Downloads != 12345 {
		t.Fatalf("expected downloads=12345, got %d", subtitles[0].Downloads)
	}

	if subtitles[1].Language != "en" {
		t.Fatalf("expected language fallback to ISO639, got %q", subtitles[1].Language)
	}
}

func TestListSubtitlesByEpisodeParsesFixture(t *testing.T) {
	fixture := loadFixture(t, "subtitles_episode.xml")

	client := newFixtureClient(t, map[string]string{
		"/en/search/sublanguageid-eng/imdbid-959621/xml": fixture,
	}, nil)

	episode := model.Episode{IMDBID: 959621, Season: 1, Number: 1}
	subtitles, err := client.ListSubtitlesByEpisode(context.Background(), episode, "eng")
	if err != nil {
		t.Fatalf("expected episode subtitle list to succeed, got: %v", err)
	}

	if len(subtitles) != 1 {
		t.Fatalf("expected 1 valid subtitle, got %d", len(subtitles))
	}

	if subtitles[0].DownloadURL != "https://dl.opensubtitles.org/en/download/subad/10991166" {
		t.Fatalf("expected absolute download url unchanged, got %q", subtitles[0].DownloadURL)
	}
}

func newFixtureClient(t *testing.T, fixtures map[string]string, assertRequest func(r *http.Request)) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if assertRequest != nil {
			assertRequest(r)
		}

		body, ok := fixtures[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}

		if filepath.Ext(r.URL.Path) == ".json" || r.URL.Path == "/libs/suggest.php" {
			w.Header().Set("Content-Type", "application/json")
		} else {
			w.Header().Set("Content-Type", "application/xml")
		}

		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
		userAgent:  defaultUserAgent,
		maxRetries: 0,
		retryDelay: 1 * time.Millisecond,
	}
}

func loadFixture(t *testing.T, fileName string) string {
	t.Helper()

	path := filepath.Join("testdata", fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fileName, err)
	}

	return string(data)
}

func assertEpisode(t *testing.T, episode model.Episode, season int, number int, imdbID int64) {
	t.Helper()

	if episode.Season != season || episode.Number != number {
		t.Fatalf("unexpected episode order: got S%02dE%02d, expected S%02dE%02d", episode.Season, episode.Number, season, number)
	}

	if episode.IMDBID != imdbID {
		t.Fatalf("unexpected episode imdb id: got %d, expected %d", episode.IMDBID, imdbID)
	}
}
