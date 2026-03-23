# sub

CLI tool to search and download subtitles from [OpenSubtitles](https://www.opensubtitles.com), straight from the terminal.

## Current MVP Behavior

- Search starts when query has at least 2 characters.
- Search requests use a 500ms debounce.
- Search suggestions come from OpenSubtitles `suggest.php`.
- Navigation uses arrow keys (`up`/`down`) and `enter`.
- Lists render with a sliding window of at most 5 visible items.
- TV flow is: title -> season -> episode -> subtitles.
- Movie flow is: title -> subtitles.
- Subtitle download uses OpenSubtitles `LinkDownload` URLs.
- Downloaded archives are extracted to a fixed output directory (`~/Downloads/sub`).

## Requirements

- [Go](https://go.dev/) 1.25+

## Getting started

```bash
# run directly
make run

# or build and run the binary
make build
./bin/sub
```

## Project structure

```
cmd/sub/          → CLI entrypoint
internal/
  model/          → domain entities (Title, Episode, Subtitle, etc.)
  service/        → business logic and adapter contracts
  opensubtitles/  → OpenSubtitles API adapter
  store/          → local persistence (save/extract files)
  ui/             → interactive terminal UI (Bubble Tea)
```

## Useful commands

```bash
make run      # run the CLI
make build    # build binary to ./bin
make test     # run tests
make check    # fmt + vet + test
make clean    # remove build artifacts
```

## Deferred Configuration (Post-MVP)

- Configurable subtitle language (currently fixed to `eng`).
- Configurable download directory (currently fixed to `~/Downloads/sub`).
- User-defined ranking/selection preferences.
- Optional archive retention policy (keep/remove zip after extraction).
- Multi-provider subtitle backends.
