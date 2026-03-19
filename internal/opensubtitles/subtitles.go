package opensubtitles

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"

	"sub/internal/model"
)

type subtitleSearchXML struct {
	Search struct {
		Results subtitleResultsXML `xml:"results"`
	} `xml:"search"`
}

type subtitleResultsXML struct {
	Subtitles []subtitleNodeXML `xml:"subtitle"`
}

type subtitleNodeXML struct {
	IDSubtitle      idSubtitleXML    `xml:"IDSubtitle"`
	ISO639          string           `xml:"ISO639"`
	LanguageName    string           `xml:"LanguageName"`
	SubFormat       string           `xml:"SubFormat"`
	MovieRelease    string           `xml:"MovieReleaseName"`
	SubFromTrusted  int              `xml:"SubFromTrusted"`
	SubDownloadsCnt textValueXML     `xml:"SubDownloadsCnt"`
	SubRating       textValueXML     `xml:"SubRating"`
	SubAddDate      textValueXML     `xml:"SubAddDate"`
	SeriesSeason    int              `xml:"SeriesSeason"`
	SeriesEpisode   int              `xml:"SeriesEpisode"`
	EpisodeName     linkTextValueXML `xml:"EpisodeName"`
}

type idSubtitleXML struct {
	Value        string `xml:",chardata"`
	LinkDownload string `xml:"LinkDownload,attr"`
}

type textValueXML struct {
	Value string `xml:",chardata"`
}

func (c *Client) ListSubtitlesByTitle(ctx context.Context, title model.Title, language string) ([]model.Subtitle, error) {
	if title.IDMovie == 0 {
		return nil, fmt.Errorf("title idmovie is required")
	}

	endpoint := fmt.Sprintf("/en/search/%s/idmovie-%d/xml", languagePath(language), title.IDMovie)
	return c.listSubtitles(ctx, endpoint)
}

func (c *Client) ListSubtitlesByEpisode(ctx context.Context, episode model.Episode, language string) ([]model.Subtitle, error) {
	if episode.IMDBID == 0 {
		return nil, fmt.Errorf("episode imdbid is required")
	}

	endpoint := fmt.Sprintf("/en/search/%s/imdbid-%d/xml", languagePath(language), episode.IMDBID)
	return c.listSubtitles(ctx, endpoint)
}

func (c *Client) listSubtitles(ctx context.Context, endpoint string) ([]model.Subtitle, error) {
	body, err := c.get(ctx, endpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("fetch subtitles list: %w", err)
	}

	var payload subtitleSearchXML
	if err := xml.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode subtitles xml: %w", err)
	}

	results := make([]model.Subtitle, 0, len(payload.Search.Results.Subtitles))
	for _, item := range payload.Search.Results.Subtitles {
		subtitleID := parseInt64(item.IDSubtitle.Value)
		downloadURL := resolveURL(c.baseURL, item.IDSubtitle.LinkDownload)
		if subtitleID == 0 || downloadURL == "" {
			continue
		}

		language := item.LanguageName
		if language == "" {
			language = item.ISO639
		}

		results = append(results, model.Subtitle{
			ID:          subtitleID,
			Language:    language,
			Format:      item.SubFormat,
			DownloadURL: downloadURL,
			ReleaseName: item.MovieRelease,
			Trusted:     item.SubFromTrusted == 1,
			Downloads:   parseInt64(item.SubDownloadsCnt.Value),
			Rating:      parseFloat64(item.SubRating.Value),
		})
	}

	return results, nil
}
