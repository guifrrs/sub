package model

type TitleType string

const (
	TitleTypeMovie  TitleType = "movie"
	TitleTypeSeries TitleType = "series"
)

type Title struct {
	IDMovie int64
	IMDBID  int64
	Name    string
	Year    int
	Type    TitleType
}

func (t Title) IsMovie() bool {
	return t.Type == TitleTypeMovie
}

func (t Title) IsSeries() bool {
	return t.Type == TitleTypeSeries
}
