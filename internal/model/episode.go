package model

type Episode struct {
	IDMovie      int64
	IMDBID       int64
	SeriesIMDBID int64
	SeriesName   string
	EpisodeName  string
	Season       int
	Number       int
}
