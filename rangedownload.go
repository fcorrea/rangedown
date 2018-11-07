package rangedownload

import (
	"sync"
)

const MaxConcurrentDownloads = 16

// FileChunk holds the information regarding the download of a file
type FileChunk struct {
	ID        int
	FileName  string
	ChunkSize int64
}

// RangeDownload holds information about a download that is performed in paralell
// using the 'Accept-Ranges' http header if supported by the server
type RangeDownload struct {
	Chunks              []*FileChunk
	TotalSize           int64
	ConcurrentDownloads int
	TotalProgress       int
	wg                  sync.WaitGroup
}

// Download will download and write the content to a temporary file
func (f *FileChunk) Download(wg *sync.WaitGroup, chn chan int) error {
	defer wg.Done()
	chn <- 0
	return nil
}

// NewRangeDownlaod initializes a RangeDownload struct with the url
func NewRangeDownlaod(u string) *RangeDownload {
	r := &RangeDownload{}
	return r
}

// init ensures that code can execute
func (r *RangeDownload) init() error {
	if r.ConcurrentDownloads > MaxConcurrentDownloads {
		r.ConcurrentDownloads = MaxConcurrentDownloads
	}
	s := r.TotalSize / int64(r.ConcurrentDownloads)
	for i := 0; i < r.ConcurrentDownloads; i++ {
		chunk := &FileChunk{
			ChunkSize: s,
		}
		r.Chunks = append(r.Chunks, chunk)
	}
	return nil
}

// Read will spawn a few goroutines that download the file in parallel
func (r *RangeDownload) Read(buf []byte) (int, error) {
	for _, f := range r.Chunks {
		p := make(chan int)
		go f.Download(&r.wg, p)
	}
	return 0, nil
}

// GetRanges will create a map containing the download ranges for all chunks
func (f *RangeDownload) GetRanges() map[int][]int64 {
	chunkSize := f.TotalSize / int64(len(f.Chunks))
	remainder := f.TotalSize % int64(len(f.Chunks))
	chunks := len(f.Chunks) - 1 // reserve a slot for the remainder bytes

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
