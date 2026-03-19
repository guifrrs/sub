package opensubtitles

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"sort"

	"sub/internal/model"
)

type seriesSearchXML struct {
	Search struct {
		Results seriesResultsXML `xml:"results"`
	} `xml:"search"`
}

type seriesResultsXML struct {
	Subtitles []seriesEpisodeNodeXML `xml:"subtitle"`
}

type seriesEpisodeNodeXML struct {
	MovieName    string           `xml:"MovieName"`
	EpisodeName  linkTextValueXML `xml:"EpisodeName"`
	SeriesSeason int              `xml:"SeriesSeason"`
	SeriesEp     int              `xml:"SeriesEpisode"`
}

type linkTextValueXML struct {
	Value string `xml:",chardata"`
	Link  string `xml:"Link,attr"`
}

func (c *Client) ListEpisodes(ctx context.Context, series model.Title, language string) ([]model.Episode, error) {
	if series.IDMovie == 0 {
		return nil, fmt.Errorf("series idmovie is required")
	}

	endpoint := fmt.Sprintf("/en/ssearch/%s/idmovie-%d/xml", languagePath(language), series.IDMovie)
	body, err := c.get(ctx, endpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("fetch series episodes: %w", err)
	}

	var payload seriesSearchXML
	if err := xml.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode series xml: %w", err)
	}

	episodes := make([]model.Episode, 0, len(payload.Search.Results.Subtitles))
	seen := make(map[string]struct{})

	for _, raw := range payload.Search.Results.Subtitles {
		if raw.SeriesSeason == 0 || raw.SeriesEp == 0 {
			continue
		}

		key := fmt.Sprintf("%d-%d", raw.SeriesSeason, raw.SeriesEp)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		episodes = append(episodes, model.Episode{
			IMDBID:       parseIDFromPath(raw.EpisodeName.Link, "imdbid-"),
			SeriesIMDBID: series.IMDBID,
			SeriesName:   series.Name,
			EpisodeName:  raw.EpisodeName.Value,
			Season:       raw.SeriesSeason,
			Number:       raw.SeriesEp,
		})
	}

	sort.Slice(episodes, func(i, j int) bool {
		if episodes[i].Season == episodes[j].Season {
			return episodes[i].Number < episodes[j].Number
		}

		return episodes[i].Season < episodes[j].Season
	})

	return episodes, nil
}
