package model

type Subtitle struct {
	ID          int64
	Language    string
	Format      string
	DownloadURL string
	ReleaseName string
	Trusted     bool
	Downloads   int64
	Rating      float64
}
