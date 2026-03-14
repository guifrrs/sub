package service

import (
	"context"
	"io"

	"sub/internal/model"
)

type Catalog interface {
	SearchTitles(ctx context.Context, query string, language string) ([]model.Title, error)
	ListEpisodes(ctx context.Context, series model.Title, language string) ([]model.Episode, error)
	ListSubtitlesByTitle(ctx context.Context, title model.Title, language string) ([]model.Subtitle, error)
	ListSubtitlesByEpisode(ctx context.Context, episode model.Episode, language string) ([]model.Subtitle, error)
}

type Downloader interface {
	DownloadArchive(ctx context.Context, downloadURL string) (archive io.ReadCloser, fileName string, err error)
}

type Store interface {
	EnsureDir(path string) error
	SaveArchive(ctx context.Context, archive io.Reader, fileName string, dir string) (archivePath string, err error)
	ExtractSubtitles(ctx context.Context, archivePath string, outputDir string) ([]string, error)
}
