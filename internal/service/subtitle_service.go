package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"sub/internal/model"
)

type SubtitleService struct {
	catalog    Catalog
	downloader Downloader
	store      Store
	language   string
	outputDir  string
}

func NewSubtitleService(catalog Catalog, downloader Downloader, store Store) *SubtitleService {
	return &SubtitleService{
		catalog:    catalog,
		downloader: downloader,
		store:      store,
		language:   DefaultLanguage,
		outputDir:  DefaultDownloadDir(),
	}
}

func (s *SubtitleService) SearchTitles(ctx context.Context, query string) ([]model.Title, error) {
	trimmed := strings.TrimSpace(query)
	if len(trimmed) < MinSearchQueryLength {
		return []model.Title{}, nil
	}

	if s.catalog == nil {
		return nil, fmt.Errorf("catalog adapter is not configured")
	}

	titles, err := s.catalog.SearchTitles(ctx, trimmed, s.language)
	if err != nil {
		return nil, fmt.Errorf("search titles: %w", err)
	}

	return titles, nil
}

func (s *SubtitleService) ListEpisodes(ctx context.Context, series model.Title) ([]model.Episode, error) {
	if !series.IsSeries() {
		return nil, fmt.Errorf("title %q is not a series", series.Name)
	}

	if s.catalog == nil {
		return nil, fmt.Errorf("catalog adapter is not configured")
	}

	episodes, err := s.catalog.ListEpisodes(ctx, series, s.language)
	if err != nil {
		return nil, fmt.Errorf("list episodes: %w", err)
	}

	return episodes, nil
}

func (s *SubtitleService) ListSubtitlesForTitle(ctx context.Context, title model.Title) ([]model.Subtitle, error) {
	if s.catalog == nil {
		return nil, fmt.Errorf("catalog adapter is not configured")
	}

	subtitles, err := s.catalog.ListSubtitlesByTitle(ctx, title, s.language)
	if err != nil {
		return nil, fmt.Errorf("list subtitles for title: %w", err)
	}

	return subtitles, nil
}

func (s *SubtitleService) ListSubtitlesForEpisode(ctx context.Context, episode model.Episode) ([]model.Subtitle, error) {
	if s.catalog == nil {
		return nil, fmt.Errorf("catalog adapter is not configured")
	}

	subtitles, err := s.catalog.ListSubtitlesByEpisode(ctx, episode, s.language)
	if err != nil {
		return nil, fmt.Errorf("list subtitles for episode: %w", err)
	}

	return subtitles, nil
}

func (s *SubtitleService) DownloadSubtitle(ctx context.Context, subtitle model.Subtitle) (model.DownloadResult, error) {
	if subtitle.DownloadURL == "" {
		return model.DownloadResult{}, fmt.Errorf("subtitle download url is empty")
	}

	if s.downloader == nil {
		return model.DownloadResult{}, fmt.Errorf("downloader adapter is not configured")
	}

	if s.store == nil {
		return model.DownloadResult{}, fmt.Errorf("store adapter is not configured")
	}

	if err := s.store.EnsureDir(s.outputDir); err != nil {
		return model.DownloadResult{}, fmt.Errorf("prepare output directory: %w", err)
	}

	archive, fileName, err := s.downloader.DownloadArchive(ctx, subtitle.DownloadURL)
	if err != nil {
		return model.DownloadResult{}, fmt.Errorf("download subtitle archive: %w", err)
	}
	defer archive.Close()

	archivePath, err := s.store.SaveArchive(ctx, archive, fileName, s.outputDir)
	if err != nil {
		return model.DownloadResult{}, fmt.Errorf("save archive: %w", err)
	}

	extractedPaths, err := s.store.ExtractSubtitles(ctx, archivePath, s.outputDir)
	if err != nil {
		return model.DownloadResult{}, fmt.Errorf("extract subtitle files: %w", err)
	}

	return model.DownloadResult{
		OutputDir:      s.outputDir,
		ArchiveName:    filepath.Base(archivePath),
		ExtractedPaths: extractedPaths,
	}, nil
}

func (s *SubtitleService) OutputDir() string {
	return s.outputDir
}
