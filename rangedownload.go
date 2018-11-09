package rangedownload

type RangeDownload struct {
	URL string
}

func NewRangeDownload(url string) *RangeDownload {
	return &RangeDownload{
		URL: url,
	}
}
