package rangedownload

import (
	"io"
	"sync"

	"github.com/jeffallen/seekinghttp"
)

const MaxConcurrentDownloads = 16

// FileChunk holds the information regarding the download of a file
type FileChunk struct {
	ID        int
	FileName  string
	ChunkSize int64
}

// Rangedownload wrapps a seekinghttp.Seekinghttp field that implements
// io.ReadSeeker and io.ReaderAt interfaces, which is needed for a range download
type RangeDownload struct {
	Chunks              []*FileChunk
	SeekingHTTP         seekinghttp.SeekingHTTP
	TotalSize           int64
	ConcurrentDownloads int
	TotalProgress       int
	wg                  sync.WaitGroup
}

// Compile-time check of interface implementations.
var _ io.ReadSeeker = (*RangeDownload)(nil)
var _ io.ReaderAt = (*RangeDownload)(nil)

// Download will download and write the content to a temporary file
func (f *FileChunk) Download(wg *sync.WaitGroup, chn chan int) error {
	defer wg.Done()
	chn <- 0
	return nil
}

// NewRangeDownlaod initializes a RangeDownload struct with the url
func NewRangeDownlaod(u string) *RangeDownload {
	return &RangeDownload{}
}

// init ensures that code can execute
func (r *RangeDownload) init() error {
	if r.TotalSize == 0 {
		size, err := r.SeekingHTTP.Size()
		if err != nil {
			r.TotalSize = size
		}
	}
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

// ReaderAt wraps seekinghttp.Seekinghttp.ReaderAt so RangeDownload can
// also be compliant with io.ReaderAt
func (r *RangeDownload) ReadAt(p []byte, off int64) (int, error) {
	return r.SeekingHTTP.ReadAt(p, off)
}

// Read will spawn a few goroutines that download the file in parallel
func (r *RangeDownload) Read(buf []byte) (int, error) {
	for _, f := range r.Chunks {
		p := make(chan int)
		go f.Download(&r.wg, p)
	}
	return 0, nil
}

func (r *RangeDownload) Seek(offset int64, whence int) (int64, error) {
	return r.SeekingHTTP.Seek(offset, whence)
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
