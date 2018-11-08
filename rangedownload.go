package rangedownload

import (
	"sync"
)

const MaxConcurrentDownloads = 16

// FileChunk holds the information regarding the download of a file
type FileChunk struct {
	URL       string
	ID        int
	FileName  string
	ChunkSize int64
}

// RangeDownload holds information about a download that is performed in paralell
// using the 'Accept-Ranges' http header if supported by the server
type RangeDownload struct {
	URL                 string
	DestinationDir      string
	Chunks              []*FileChunk
	Size                int64
	ConcurrentDownloads int
	Progress            int
	wg                  sync.WaitGroup
	downloaded          chan int
}

// Download will download and write the content to a temporary file
func (f *FileChunk) Download(wg *sync.WaitGroup, out chan<- int64, start, end int64) error {
	defer wg.Done()
	out <- 0
	return nil
}

// NewRangeDownlaod initializes a RangeDownload struct with the url
func NewRangeDownlaod(u string, c int, d string) *RangeDownload {
	rd := &RangeDownload{
		URL:                 u,
		DestinationDir:      d,
		ConcurrentDownloads: c,
	}
	for i := 0; i < c; i++ {
		fc := &FileChunk{
			URL: u,
			ID:  i,
		}
		rd.Chunks = append(rd.Chunks, fc)
	}
	return rd
}

// GetRanges will create a map containing the download ranges for all chunks
func (r *RangeDownload) GetRanges() map[int][]int64 {
	chunkSize := r.Size / int64(len(r.Chunks))
	remainder := r.Size % int64(len(r.Chunks))
	chunks := len(r.Chunks) - 1 // reserve a slot for the remainder bytes

	ranges := make(map[int][]int64, chunks)
	var from, to int64

	for i := 0; i < chunks; i++ {
		to += chunkSize
		ranges[i] = append(ranges[i], from, to)
		from = to + 1
	}

	// Insert the remainder bytes at the end of the map
	to = to + chunkSize + remainder
	ranges[chunks] = append(ranges[chunks], from, to)
	return ranges
}

func (r *RangeDownload) Start() error {
	chn := make(chan int64)
	ranges := r.GetRanges()
	for i := 0; i < len(r.Chunks); i++ {
		start, end := ranges[i][0], ranges[i][0]
		r.Chunks[i].Download(&r.wg, chn, start, end)
	}
	return nil
}
