# sub

CLI tool to search and download subtitles from [OpenSubtitles](https://www.opensubtitles.com), straight from the terminal.

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
