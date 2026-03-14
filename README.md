# sub CLI (MVP)

`sub` is a Go terminal app for searching and downloading subtitles from OpenSubtitles.

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

## Run

```bash
make run
```

## Verify

```bash
make test
make vet
```

## Manual End-to-End Checks (Phase 6.3)

Executed on 2026-03-14 against live OpenSubtitles endpoints.

Movie workflow check:
- Suggest query: `breakfast`
- Selected movie ID: `idmovie-6122` (Breakfast Club)
- Subtitle entries found from XML endpoint: `42`

Series workflow check:
- Suggest query: `breaking bad`
- Selected series ID: `idmovie-31305` (Breaking Bad)
- Episode entries found from series XML endpoint: `70`
- Sample episode selected: `imdbid-959621` (Pilot)
- Subtitle entries found for sample episode: `17`

## Deferred Configuration (Post-MVP)

- Configurable subtitle language (currently fixed to `eng`).
- Configurable download directory (currently fixed to `~/Downloads/sub`).
- User-defined ranking/selection preferences.
- Optional archive retention policy (keep/remove zip after extraction).
- Multi-provider subtitle backends.
