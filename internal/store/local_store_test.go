package store

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractSubtitlesValidArchive(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "subtitles.zip")
	outputDir := filepath.Join(baseDir, "out")

	err := createZip(archivePath,
		zipEntry{name: "movie.srt", content: "hello"},
		zipEntry{name: "nested/episode.ass", content: "world"},
		zipEntry{name: "notes.txt", content: "ignore me"},
	)
	if err != nil {
		t.Fatalf("create zip fixture: %v", err)
	}

	store := NewLocalStore()
	extracted, err := store.ExtractSubtitles(context.Background(), archivePath, outputDir)
	if err != nil {
		t.Fatalf("expected extraction success, got: %v", err)
	}

	if len(extracted) != 2 {
		t.Fatalf("expected 2 extracted subtitle files, got %d", len(extracted))
	}

	for _, p := range extracted {
		if !strings.HasPrefix(p, outputDir+string(os.PathSeparator)) {
			t.Fatalf("expected extracted path under output dir, got %q", p)
		}

		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected extracted file to exist: %v", err)
		}
	}
}

func TestExtractSubtitlesRejectsTraversalPath(t *testing.T) {
	baseDir := t.TempDir()
	archivePath := filepath.Join(baseDir, "bad.zip")
	outputDir := filepath.Join(baseDir, "out")

	err := createZip(archivePath,
		zipEntry{name: "../evil.srt", content: "pwnd"},
	)
	if err != nil {
		t.Fatalf("create zip fixture: %v", err)
	}

	store := NewLocalStore()
	_, err = store.ExtractSubtitles(context.Background(), archivePath, outputDir)
	if err == nil {
		t.Fatalf("expected extraction to fail for traversal path")
	}

	if !strings.Contains(err.Error(), "unsafe archive path") {
		t.Fatalf("expected unsafe archive path error, got: %v", err)
	}
}

type zipEntry struct {
	name    string
	content string
}

func createZip(path string, entries ...zipEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, entry := range entries {
		writer, err := zw.Create(entry.name)
		if err != nil {
			_ = zw.Close()
			return err
		}

		if _, err := writer.Write([]byte(entry.content)); err != nil {
			_ = zw.Close()
			return err
		}
	}

	return zw.Close()
}
