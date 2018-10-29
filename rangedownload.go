package rangedownload

import (
	"io"
	"net/http"

	"github.com/jeffallen/seekinghttp"
)

const MaxConcurrentDownloads = 16

// FileSlice holds the information regarding the download of a file
type FileSlice struct {
	FileName           string
	Progress           int64
	LastByteDownloaded int64
	ChunkSize          int64
}

// Rangedownload wrapps a seekinghttp.Seekinghttp field that implements
// io.ReadSeeker and io.ReaderAt interfaces
type RangeDownload struct {
	Slices              []*FileSlice
	URL                 string
	SeekingHTTP         *seekinghttp.SeekingHTTP
	TotalSize           int64
	ConcurrentDownloads int
	TotalProgress       int
}

// Compile-time check of interface implementations.
var _ io.ReadSeeker = (*SeekingHTTP)(nil)
var _ io.ReaderAt = (*SeekingHTTP)(nil)

// Download will download and write the content to a temporary file
func (f *FileSlice) Download(p chan int) error {
	// CONTINUE FROM HERE
}

// New initializes a RangeDownload struct with the url
func New(url string) *RangeDownload {
	skhttp := seekinghttp.New(url)
	return &RangeDownload{
		URL:         url,
		SeekingHTTP: skhttp,
	}
}

// init ensures that code can execute
func (r *RangeDownload) init() error {
	r.SeekingHTTP.init()
	if r.TotalSize == nil {
		r.TotalSize = r.SeekingHTTP.Size()
	}
	if r.ConcurrentDownloads > MaxConcurrentDownloads {
		r.ConcurrentDownloads = MaxConcurrentDownloads
	}
	sliceSize = r.TotalSize / r.ConcurrentDownloads
	for range r.ConcurrentDownloads {
		fslice := &FileSlice{
			ChunkSize: sliceSize,
		}
		append(r.Slices, fslice)
	}
	return nil
}

// ReaderAt wraps seekinghttp.Seekinghttp.ReaderAt so RangeDownload can
// also be compliant with io.ReaderAt
func (r *RangeDownload) ReaderAt(p []byte, off int64) (int, error) {
	return r.SeekingHTTP.ReadAt(p, off)
}

// Read will spawn a few goroutines that download the file in parallel
func (r *RangeDownload) Read(buf []byte) (int, error) {
	for _, fslice := range r.Slices {
		progressChan := make(chan []int)
		go fslice.Download(progressChan)
	}
}

func (r *RangeDownload) Seek(offset int64, whence int) (int64, error) {
	return r.SeekingHTTP.Seek(offset, whence)
}
