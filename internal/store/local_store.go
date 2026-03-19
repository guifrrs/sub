package store

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStore struct{}

func NewLocalStore() *LocalStore {
	return &LocalStore{}
}

func (s *LocalStore) EnsureDir(path string) error {
	resolved, err := ResolveOutputDir(path)
	if err != nil {
		return err
	}

	info, statErr := os.Stat(resolved)
	if statErr == nil {
		if !info.IsDir() {
			return fmt.Errorf("output path is not a directory: %s", resolved)
		}
		return nil
	}

	if !os.IsNotExist(statErr) {
		return fmt.Errorf("stat output directory: %w", statErr)
	}

	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	return nil
}

func ResolveOutputDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("output directory is empty")
	}

	cleaned := filepath.Clean(path)
	resolved, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}

	return resolved, nil
}

func (s *LocalStore) SaveArchive(ctx context.Context, archive io.Reader, fileName string, dir string) (string, error) {
	if archive == nil {
		return "", fmt.Errorf("archive reader is nil")
	}

	resolvedDir, err := ResolveOutputDir(dir)
	if err != nil {
		return "", err
	}

	if err := s.EnsureDir(resolvedDir); err != nil {
		return "", err
	}

	safeName := sanitizeArchiveName(fileName)
	archivePath := filepath.Join(resolvedDir, safeName)

	f, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("create archive file: %w", err)
	}
	defer f.Close()

	if err := copyWithContext(ctx, f, archive); err != nil {
		return "", fmt.Errorf("save archive file: %w", err)
	}

	return archivePath, nil
}

func (s *LocalStore) ExtractSubtitles(ctx context.Context, archivePath string, outputDir string) ([]string, error) {
	resolvedOutputDir, err := ResolveOutputDir(outputDir)
	if err != nil {
		return nil, err
	}

	if err := s.EnsureDir(resolvedOutputDir); err != nil {
		return nil, err
	}

	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer zr.Close()

	extracted := make([]string, 0)

	for _, file := range zr.File {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("extract subtitles canceled: %w", err)
		}

		if file.FileInfo().IsDir() {
			continue
		}

		if !isSubtitleFile(file.Name) {
			continue
		}

		destinationPath, err := safeDestinationPath(resolvedOutputDir, file.Name)
		if err != nil {
			return nil, fmt.Errorf("unsafe archive path %q: %w", file.Name, err)
		}

		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			return nil, fmt.Errorf("create extraction directory: %w", err)
		}

		inFile, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open archived subtitle %q: %w", file.Name, err)
		}

		outFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			inFile.Close()
			return nil, fmt.Errorf("create extracted subtitle file: %w", err)
		}

		copyErr := copyWithContext(ctx, outFile, inFile)
		closeErr := outFile.Close()
		inCloseErr := inFile.Close()

		if copyErr != nil {
			return nil, fmt.Errorf("extract subtitle %q: %w", file.Name, copyErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("finalize subtitle file %q: %w", file.Name, closeErr)
		}
		if inCloseErr != nil {
			return nil, fmt.Errorf("close archived subtitle %q: %w", file.Name, inCloseErr)
		}

		extracted = append(extracted, destinationPath)
	}

	if len(extracted) == 0 {
		return nil, fmt.Errorf("no supported subtitle files found in archive")
	}

	return extracted, nil
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}

func safeDestinationPath(outputDir string, archiveEntry string) (string, error) {
	entry := filepath.Clean(archiveEntry)
	if entry == "." {
		return "", fmt.Errorf("empty archive path")
	}

	if filepath.IsAbs(entry) {
		return "", fmt.Errorf("absolute archive path is not allowed")
	}

	if strings.HasPrefix(entry, "..") {
		return "", fmt.Errorf("archive path escapes output directory")
	}

	joined := filepath.Join(outputDir, entry)
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolve destination path: %w", err)
	}

	outputResolved, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}

	prefix := outputResolved + string(os.PathSeparator)
	if resolved != outputResolved && !strings.HasPrefix(resolved, prefix) {
		return "", fmt.Errorf("archive path escapes output directory")
	}

	return resolved, nil
}

func sanitizeArchiveName(fileName string) string {
	clean := strings.TrimSpace(fileName)
	if clean == "" {
		return "subtitle.zip"
	}

	clean = filepath.Base(clean)
	clean = strings.ReplaceAll(clean, string(os.PathSeparator), "_")
	if clean == "." || clean == "" {
		return "subtitle.zip"
	}

	if filepath.Ext(clean) == "" {
		clean += ".zip"
	}

	return clean
}

func isSubtitleFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".srt", ".sub", ".ass", ".ssa", ".vtt":
		return true
	default:
		return false
	}
}
