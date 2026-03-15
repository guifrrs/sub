package opensubtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"sub/internal/model"
)

const minSuggestQueryLength = 2

type suggestItem struct {
	Name   string         `json:"name"`
	Year   stringOrNumber `json:"year"`
	ID     stringOrNumber `json:"id"`
	Pic    stringOrNumber `json:"pic"`
	Kind   string         `json:"kind"`
	Rating stringOrNumber `json:"rating"`
}

type stringOrNumber string

func (v *stringOrNumber) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*v = ""
		return nil
	}

	var s string
	if err := json.Unmarshal(trimmed, &s); err == nil {
		*v = stringOrNumber(s)
		return nil
	}

	var n json.Number
	if err := json.Unmarshal(trimmed, &n); err == nil {
		*v = stringOrNumber(n.String())
		return nil
	}

	return fmt.Errorf("expected string or number, got %s", string(trimmed))
}

func (v stringOrNumber) String() string {
	return string(v)
}

func (c *Client) SearchTitles(ctx context.Context, query string, language string) ([]model.Title, error) {
	query = strings.TrimSpace(query)
	if len(query) < minSuggestQueryLength {
		return []model.Title{}, nil
	}

	params := url.Values{}
	params.Set("format", "json3")
	params.Set("MovieName", query)
	params.Set("SubLanguageID", normalizeLanguage(language))

	body, err := c.get(ctx, "/libs/suggest.php", params)
	if err != nil {
		return nil, fmt.Errorf("search title suggestions: %w", err)
	}

	var rawItems []suggestItem
	if err := json.Unmarshal(body, &rawItems); err != nil {
		return nil, fmt.Errorf("decode suggest response: %w", err)
	}

	titles := make([]model.Title, 0, len(rawItems))
	for _, item := range rawItems {
		typeName := strings.ToLower(strings.TrimSpace(item.Kind))
		titleType := model.TitleTypeMovie
		if typeName == "tv" || typeName == "series" || typeName == "tv series" {
			titleType = model.TitleTypeSeries
		}

		titles = append(titles, model.Title{
			IDMovie: parseInt64(item.ID.String()),
			IMDBID:  parseInt64(item.Pic.String()),
			Name:    strings.TrimSpace(item.Name),
			Year:    parseInt(item.Year.String()),
			Type:    titleType,
		})
	}

	return titles, nil
}
